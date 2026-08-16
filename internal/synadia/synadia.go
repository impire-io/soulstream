// Package synadia is the Synadia Cloud account-half driver (design 0003
// §5): the same split as the self-hosted kit, applied by an API instead
// of a human. It ensures the two accounts, the programmatic signing-key
// groups (their seeds are returned exactly once — this driver is that
// once), the callout wiring, and the issuer user under an on-demand
// group. Every step is idempotent by name; a seed that was returned to
// a previous run and lost is a named refusal, never a silent rotation.
//
// Graduated from soulstream-mcp's cmd/byon-setup (spec 018 Q2 named it
// best-effort operator tooling; journey 0038 measured the wiring live).
package synadia

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/synadia-io/control-plane-sdk-go/syncp"
)

// Config is one founding's driver input. Token is the Cloud PAT — it
// arrives by environment, is used for the account half, and is never
// persisted (design 0003 §5).
type Config struct {
	BaseURL string // default https://cloud.synadia.com
	Token   string
	System  string // the Cloud system's name (exact match, case-insensitive)
	Realm   string // names the accounts: soulstream-<realm>, soulstream-<realm>-auth

	// Existing carries seeds a previous run already custodied, so
	// re-runs recognize their own groups instead of refusing them.
	Existing Existing

	// Log receives the driver's progress lines (the founding run's
	// stderr). Optional.
	Log io.Writer
}

// Existing is the seed material a resumed founding already holds.
type Existing struct {
	RealmScopedSeed []byte
	WorkloadSeed    []byte
	AuthSigningSeed []byte
}

// Result is what the account half hands back: the two account public
// keys, the three signing-key seeds, the issuer's downloaded creds, and
// the platform's callout xkey when it set one.
type Result struct {
	AuthAccountPub  string
	RealmAccountPub string
	RealmScopedSeed []byte
	WorkloadSeed    []byte
	AuthSigningSeed []byte
	IssuerCreds     []byte

	// XkeyPublic is the platform-set callout xkey ("" when unset). When
	// set, requests are sealed to a key whose seed the platform
	// custodies — the callout then runs unsealed on our side, said out
	// loud at founding (design 0003 §5, the measured 0038 caveat).
	XkeyPublic string
}

const (
	groupScoped   = "soulstream-user"
	groupWorkload = "soulstream-workload"
	groupAuth     = "soulstream-auth-issuer"
	groupOnDemand = "soulstream-auth-users"
	issuerUser    = "soulstream-identity-issuer"
)

// Setup runs the account half. scopePub and scopeSub are the persona
// scope's allow lists (ceremony.PersonaScopeAllows — one source, so the
// Cloud-authored scope cannot drift from the embedded one).
func Setup(ctx context.Context, cfg Config, scopePub, scopeSub []string) (*Result, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("synadia: an API token is required — export SOULSTREAM_SYNADIA_TOKEN (mint one under Personal Access Tokens in Synadia Cloud)")
	}
	if cfg.System == "" {
		return nil, fmt.Errorf("synadia: the system name is required (--synadia-system)")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://cloud.synadia.com"
	}
	logf := func(format string, args ...any) {
		if cfg.Log != nil {
			fmt.Fprintf(cfg.Log, "soulstream: synadia: "+format+"\n", args...)
		}
	}
	client := syncp.NewAPIClient(syncp.NewConfiguration())
	ctx = context.WithValue(ctx, syncp.ContextServerVariables, map[string]string{"baseUrl": base})
	ctx = context.WithValue(ctx, syncp.ContextAccessToken, cfg.Token)

	// Team → system, by exact (case-insensitive) name.
	teams, _, err := client.SessionAPI.ListTeams(ctx).Execute()
	if err != nil {
		return nil, apiErr("list teams", err)
	}
	var system *syncp.SystemViewResponse
	for _, team := range teams.Items {
		systems, _, err := client.TeamAPI.ListTeamSystems(ctx, team.Id).Execute()
		if err != nil {
			return nil, apiErr("list systems for team "+team.Name, err)
		}
		for i, s := range systems.Items {
			if strings.EqualFold(s.Name, cfg.System) || s.Id == cfg.System {
				system = &systems.Items[i]
			}
		}
	}
	if system == nil {
		return nil, fmt.Errorf("synadia: no system named %q on this token's teams (the name or the system id)", cfg.System)
	}
	logf("system %q (id %s)", system.Name, system.Id)

	// The two accounts, created when missing — and nothing else touched
	// (design 0003 §2: the driver's scope is the two accounts and their
	// keys).
	realmName := "soulstream-" + cfg.Realm
	authName := realmName + "-auth"
	accounts, _, err := client.SystemAPI.ListAccounts(ctx, system.Id).Execute()
	if err != nil {
		return nil, apiErr("list accounts", err)
	}
	var realmAcct, authAcct *syncp.AccountViewResponse
	for i, a := range accounts.Items {
		switch {
		case strings.EqualFold(a.Name, realmName):
			realmAcct = &accounts.Items[i]
		case strings.EqualFold(a.Name, authName):
			authAcct = &accounts.Items[i]
		}
	}
	ensureAccount := func(existing *syncp.AccountViewResponse, name string) (*syncp.AccountViewResponse, error) {
		if existing != nil {
			logf("account %q exists (id %s)", name, existing.Id)
			return existing, nil
		}
		created, _, err := client.SystemAPI.CreateAccount(ctx, system.Id).
			AccountCreateRequest(syncp.AccountCreateRequest{Name: name}).Execute()
		if err != nil {
			return nil, apiErr("create account "+name, err)
		}
		logf("created account %q (id %s)", name, created.Id)
		return created, nil
	}
	if realmAcct, err = ensureAccount(realmAcct, realmName); err != nil {
		return nil, err
	}
	if authAcct, err = ensureAccount(authAcct, authName); err != nil {
		return nil, err
	}

	res := &Result{
		RealmAccountPub: deref(realmAcct.AccountPublicKey),
		AuthAccountPub:  deref(authAcct.AccountPublicKey),
	}
	if res.RealmAccountPub == "" || res.AuthAccountPub == "" {
		return nil, fmt.Errorf("synadia: the platform returned no public key for %q or %q — cannot hand back the account half", realmName, authName)
	}

	// The three signing-key groups. Programmatic groups return their
	// seed exactly once (measured, journey 0038): a group that exists
	// while we hold no seed is unrecoverable and refused by name.
	scope := &syncp.UserPermissionLimits{Permissions: syncp.Permissions{
		Pub: &syncp.Permission{Allow: scopePub},
		Sub: &syncp.Permission{Allow: scopeSub},
	}}
	scopedID, scopedSeed, err := ensureGroup(ctx, client, logf, realmAcct.Id, groupScoped, scope, cfg.Existing.RealmScopedSeed)
	if err != nil {
		return nil, err
	}
	res.RealmScopedSeed = scopedSeed
	if _, res.WorkloadSeed, err = ensureGroup(ctx, client, logf, realmAcct.Id, groupWorkload, nil, cfg.Existing.WorkloadSeed); err != nil {
		return nil, err
	}
	authGroupID, authSeed, err := ensureGroup(ctx, client, logf, authAcct.Id, groupAuth, nil, cfg.Existing.AuthSigningSeed)
	if err != nil {
		return nil, err
	}
	res.AuthSigningSeed = authSeed

	// Callout: enabled for the system with the AUTH account as control
	// account; tolerate "already enabled" — re-runs must not fail here
	// (journey 0038's wiring).
	if _, err := client.SystemAPI.EnableAuthCallout(ctx, system.Id).
		AuthCalloutEnableRequest(syncp.AuthCalloutEnableRequest{ControlAccount: authAcct.Id}).Execute(); err != nil {
		logf("enable auth callout: %v (continuing — may already be enabled)", err)
	} else {
		logf("auth callout enabled (control account %s)", authName)
	}
	configs, _, err := client.SystemAPI.ListAuthCalloutConfigs(ctx, system.Id).Execute()
	if err != nil {
		return nil, apiErr("list auth callout configs", err)
	}
	calloutID := ""
	for _, c := range configs.Items {
		if c.ControlAccountId == authAcct.Id {
			calloutID = c.Id
			res.XkeyPublic = deref(c.XkeyPublic)
		}
	}
	if calloutID == "" {
		return nil, fmt.Errorf("synadia: no callout config for control account %q after enabling — the platform did not wire it", authName)
	}
	if _, err := client.AuthCalloutAPI.AddAuthCalloutTargetAccount(ctx, calloutID).
		AuthCalloutAddTargetAccountRequest(syncp.AuthCalloutAddTargetAccountRequest{
			AccountId:               realmAcct.Id,
			SkGroupId:               scopedID,
			ControlAccountSkGroupId: authGroupID,
		}).Execute(); err != nil {
		logf("add target account: %v (continuing — may already be wired)", err)
	} else {
		logf("target account wired: %s admits through the callout", realmName)
	}

	// The issuer user: the platform refuses NATS users under
	// programmatic groups (measured, journey 0038), so it lives under an
	// on-demand group of its own; registering it as a callout user puts
	// it in auth_users, and its downloaded creds are the issuer
	// connection.
	onDemandID, err := ensureOnDemandGroup(ctx, client, logf, authAcct.Id, groupOnDemand)
	if err != nil {
		return nil, err
	}
	issuerID, err := ensureUser(ctx, client, logf, authAcct.Id, issuerUser, onDemandID)
	if err != nil {
		return nil, err
	}
	if _, err := client.AuthCalloutAPI.AddAuthCalloutUser(ctx, calloutID).
		AuthCalloutAddUserRequest(syncp.AuthCalloutAddUserRequest{NatsUserId: issuerID}).Execute(); err != nil {
		logf("add callout user: %v (continuing — may already be registered)", err)
	}
	creds, _, err := client.NatsUserAPI.DownloadNatsUserCreds(ctx, issuerID).Execute()
	if err != nil {
		return nil, apiErr("download issuer creds", err)
	}
	res.IssuerCreds = []byte(creds)
	return res, nil
}

// ensureGroup finds a programmatic signing-key group by name or creates
// it. Created: the once-returned seed comes back. Found: the caller's
// existing seed must cover it, or the group is refused by name.
func ensureGroup(ctx context.Context, client *syncp.APIClient, logf func(string, ...any),
	accountID, name string, scope *syncp.UserPermissionLimits, existingSeed []byte) (string, []byte, error) {
	groups, _, err := client.AccountAPI.ListAccountSkGroup(ctx, accountID).Execute()
	if err != nil {
		return "", nil, apiErr("list signing-key groups", err)
	}
	for _, g := range groups.Items {
		if g.Name != name {
			continue
		}
		if len(existingSeed) == 0 {
			return "", nil, fmt.Errorf("synadia: signing-key group %q already exists and its seed was returned to a previous run — a programmatic seed is shown exactly once. Delete the group in Synadia Cloud (or found into a fresh realm name) and re-run", name)
		}
		logf("signing-key group %q exists (id %s) — reusing its custodied seed", name, g.Id)
		return g.Id, existingSeed, nil
	}
	req := syncp.SigningKeyGroupCreateRequest{Name: name, Programmatic: true}
	if scope != nil {
		req.Scope = scope
	}
	created, _, err := client.AccountAPI.CreateAccountSkGroup(ctx, accountID).
		SigningKeyGroupCreateRequest(req).Execute()
	if err != nil {
		return "", nil, apiErr("create signing-key group "+name, err)
	}
	if created.Seed == nil || *created.Seed == "" {
		return "", nil, fmt.Errorf("synadia: signing-key group %q was created but no seed came back — cannot custody it", name)
	}
	logf("created signing-key group %q (id %s); seed custodied", name, created.Id)
	return created.Id, []byte(*created.Seed), nil
}

// ensureOnDemandGroup finds or creates a NON-programmatic group — the
// kind the platform allows NATS users under (its seed stays with
// Synadia).
func ensureOnDemandGroup(ctx context.Context, client *syncp.APIClient, logf func(string, ...any),
	accountID, name string) (string, error) {
	groups, _, err := client.AccountAPI.ListAccountSkGroup(ctx, accountID).Execute()
	if err != nil {
		return "", apiErr("list signing-key groups", err)
	}
	for _, g := range groups.Items {
		if g.Name == name && !g.Programmatic {
			logf("on-demand signing-key group %q exists (id %s)", name, g.Id)
			return g.Id, nil
		}
	}
	created, _, err := client.AccountAPI.CreateAccountSkGroup(ctx, accountID).
		SigningKeyGroupCreateRequest(syncp.SigningKeyGroupCreateRequest{Name: name, Programmatic: false}).Execute()
	if err != nil {
		return "", apiErr("create on-demand signing-key group "+name, err)
	}
	logf("created on-demand signing-key group %q (id %s)", name, created.Id)
	return created.Id, nil
}

func ensureUser(ctx context.Context, client *syncp.APIClient, logf func(string, ...any),
	accountID, name, skGroupID string) (string, error) {
	users, _, err := client.AccountAPI.ListUsers(ctx, accountID).Execute()
	if err != nil {
		return "", apiErr("list nats users", err)
	}
	for _, u := range users.Items {
		if u.Name == name {
			logf("nats user %q exists (id %s)", name, u.Id)
			return u.Id, nil
		}
	}
	created, _, err := client.AccountAPI.CreateUser(ctx, accountID).
		NatsUserCreateRequest(syncp.NatsUserCreateRequest{Name: name, SkGroupId: skGroupID}).Execute()
	if err != nil {
		return "", apiErr("create nats user "+name, err)
	}
	logf("created nats user %q (id %s)", name, created.Id)
	return created.Id, nil
}

func apiErr(what string, err error) error {
	if apiE, ok := err.(*syncp.GenericOpenAPIError); ok {
		return fmt.Errorf("synadia: %s: %w — %s", what, err, string(apiE.Body()))
	}
	return fmt.Errorf("synadia: %s: %w", what, err)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
