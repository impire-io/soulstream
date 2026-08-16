// Package synadia is the Synadia Cloud account-half driver (design 0003
// §5): the same split as the self-hosted kit, applied by an API instead
// of a human. It ensures the two accounts, the programmatic signing-key
// groups (their seeds are returned exactly once — this driver is that
// once), the callout wiring, and the issuer user under an on-demand
// group. Every step is idempotent by name; a seed that was returned to
// a previous run and lost is a named refusal, never a silent rotation.
//
// The channel to a BYON is allowed to be lossy: the platform's
// private-link tunnel was measured cycling mid-request (2026-08-16,
// ~50% of mutations drew 500 "nats: timeout"), so every step retries
// bounded through 5xx, re-listing first so a create that landed
// despite its timeout is found, never doubled.
//
// Graduated from soulstream-mcp's cmd/byon-setup (spec 018 Q2 named it
// best-effort operator tooling; journey 0038 measured the wiring live).
package synadia

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/synadia-io/control-plane-sdk-go/syncp"
)

// Config is one founding's driver input. Token is the Cloud PAT — it
// arrives by environment, is used for the account half, and is never
// persisted (design 0003 §5).
type Config struct {
	BaseURL string // default https://cloud.synadia.com
	Token   string
	System  string // the Cloud system's name or id (name case-insensitive)
	Realm   string // names the accounts: soulstream-<realm>, soulstream-<realm>-auth

	// Existing carries seeds a previous run already custodied, so
	// re-runs recognize their own groups instead of refusing them.
	Existing Existing

	// OnSeed receives each programmatic group's once-returned seed the
	// moment it arrives, BEFORE any further API call — the caller
	// persists it immediately, so a failure later in the account half
	// can never lose it (measured 2026-08-16: a mid-run tunnel cycle
	// after the scoped group's create left its seed in a dead process's
	// memory; the group had to be disabled and deleted by hand).
	// Optional, but any caller that persists state wants it.
	OnSeed func(group string, seed []byte) error

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

// The group names, exported so OnSeed callers can map a seed to its
// place in the ceremony state.
const (
	GroupScoped   = "soulstream-user"
	GroupWorkload = "soulstream-workload"
	GroupAuth     = "soulstream-auth-issuer"

	groupOnDemand = "soulstream-auth-users"
	issuerUser    = "soulstream-identity-issuer"
)

// attempts and the backoff base for lossy-channel retries. Worst case
// per step: 4 tries across ~12s of sleep — one private-link tunnel
// cycle wide. The base is a variable only so tests need not sleep.
const attempts = 4

var backoffBase = 2 * time.Second

func backoff(ctx context.Context, attempt int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(attempt) * backoffBase):
		return nil
	}
}

// is5xx reports a server-side failure worth retrying (the measured BYON
// flake surfaces as 500 {"error":"nats: timeout"}).
func is5xx(resp *http.Response) bool { return resp != nil && resp.StatusCode >= 500 }

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
	d := &driver{ctx: ctx, client: client, logf: logf, onSeed: cfg.OnSeed}

	// Team → system, by exact (case-insensitive) name or by id.
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
	realmAcct, err := d.ensureAccount(system.Id, realmName)
	if err != nil {
		return nil, err
	}
	authAcct, err := d.ensureAccount(system.Id, authName)
	if err != nil {
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
	// seed exactly once (measured, journey 0038): the seed is handed to
	// OnSeed before the next step, and a group that exists while no one
	// holds its seed is unrecoverable and refused by name.
	scope := &syncp.UserPermissionLimits{Permissions: syncp.Permissions{
		Pub: &syncp.Permission{Allow: scopePub},
		Sub: &syncp.Permission{Allow: scopeSub},
	}}
	scopedID, scopedSeed, err := d.ensureGroup(realmAcct.Id, GroupScoped, scope, cfg.Existing.RealmScopedSeed)
	if err != nil {
		return nil, err
	}
	res.RealmScopedSeed = scopedSeed
	if _, res.WorkloadSeed, err = d.ensureGroup(realmAcct.Id, GroupWorkload, nil, cfg.Existing.WorkloadSeed); err != nil {
		return nil, err
	}
	authGroupID, authSeed, err := d.ensureGroup(authAcct.Id, GroupAuth, nil, cfg.Existing.AuthSigningSeed)
	if err != nil {
		return nil, err
	}
	res.AuthSigningSeed = authSeed

	// Callout: enabled for the system with the AUTH account as control
	// account; a non-5xx error is tolerated — re-runs must not fail on
	// "already enabled" (journey 0038's wiring) — and the config list
	// below is the backstop that the enable actually holds.
	if err := d.retry("enable auth callout", func() (*http.Response, error) {
		return client.SystemAPI.EnableAuthCallout(ctx, system.Id).
			AuthCalloutEnableRequest(syncp.AuthCalloutEnableRequest{ControlAccount: authAcct.Id}).Execute()
	}, true); err != nil {
		return nil, err
	}
	var calloutID string
	if err := d.retry("list auth callout configs", func() (*http.Response, error) {
		configs, resp, err := client.SystemAPI.ListAuthCalloutConfigs(ctx, system.Id).Execute()
		if err != nil {
			return resp, err
		}
		for _, c := range configs.Items {
			if c.ControlAccountId == authAcct.Id {
				calloutID = c.Id
				res.XkeyPublic = deref(c.XkeyPublic)
			}
		}
		return resp, nil
	}, false); err != nil {
		return nil, err
	}
	if calloutID == "" {
		return nil, fmt.Errorf("synadia: no callout config for control account %q after enabling — the platform did not wire it", authName)
	}
	if err := d.retry("wire target account", func() (*http.Response, error) {
		return client.AuthCalloutAPI.AddAuthCalloutTargetAccount(ctx, calloutID).
			AuthCalloutAddTargetAccountRequest(syncp.AuthCalloutAddTargetAccountRequest{
				AccountId:               realmAcct.Id,
				SkGroupId:               scopedID,
				ControlAccountSkGroupId: authGroupID,
			}).Execute()
	}, true); err != nil {
		return nil, err
	}

	// The issuer user: the platform refuses NATS users under
	// programmatic groups (measured, journey 0038), so it lives under an
	// on-demand group of its own; registering it as a callout user puts
	// it in auth_users, and its downloaded creds are the issuer
	// connection.
	onDemandID, err := d.ensureOnDemandGroup(authAcct.Id, groupOnDemand)
	if err != nil {
		return nil, err
	}
	issuerID, err := d.ensureUser(authAcct.Id, issuerUser, onDemandID)
	if err != nil {
		return nil, err
	}
	if err := d.retry("register callout user", func() (*http.Response, error) {
		return client.AuthCalloutAPI.AddAuthCalloutUser(ctx, calloutID).
			AuthCalloutAddUserRequest(syncp.AuthCalloutAddUserRequest{NatsUserId: issuerID}).Execute()
	}, true); err != nil {
		return nil, err
	}
	if err := d.retry("download issuer creds", func() (*http.Response, error) {
		creds, resp, err := client.NatsUserAPI.DownloadNatsUserCreds(ctx, issuerID).Execute()
		if err == nil {
			res.IssuerCreds = []byte(creds)
		}
		return resp, err
	}, false); err != nil {
		return nil, err
	}
	return res, nil
}

// driver bundles the per-run plumbing the ensure helpers share.
type driver struct {
	ctx    context.Context
	client *syncp.APIClient
	logf   func(string, ...any)
	onSeed func(string, []byte) error
}

// retry runs f through the lossy channel: 5xx retries with backoff;
// other errors return (or are tolerated and logged when tolerate is
// set — the "may already be configured" class).
func (d *driver) retry(what string, f func() (*http.Response, error), tolerate bool) error {
	for attempt := 1; ; attempt++ {
		resp, err := f()
		if err == nil {
			return nil
		}
		if is5xx(resp) && attempt < attempts {
			d.logf("%s: %v — the channel to the system is lossy; retrying (%d/%d)", what, err, attempt, attempts)
			if berr := backoff(d.ctx, attempt); berr != nil {
				return apiErr(what, err)
			}
			continue
		}
		if tolerate && !is5xx(resp) {
			d.logf("%s: %v (continuing — may already be configured)", what, err)
			return nil
		}
		return apiErr(what, err)
	}
}

// ensureAccount finds an account by name or creates it, re-listing
// between attempts so a create that landed despite a timeout is found,
// never doubled.
func (d *driver) ensureAccount(systemID, name string) (*syncp.AccountViewResponse, error) {
	for attempt := 1; ; attempt++ {
		accounts, resp, err := d.client.SystemAPI.ListAccounts(d.ctx, systemID).Execute()
		if err != nil {
			if is5xx(resp) && attempt < attempts {
				if backoff(d.ctx, attempt) != nil {
					return nil, apiErr("list accounts", err)
				}
				continue
			}
			return nil, apiErr("list accounts", err)
		}
		for i, a := range accounts.Items {
			if strings.EqualFold(a.Name, name) {
				d.logf("account %q exists (id %s)", name, a.Id)
				return &accounts.Items[i], nil
			}
		}
		created, resp, err := d.client.SystemAPI.CreateAccount(d.ctx, systemID).
			AccountCreateRequest(syncp.AccountCreateRequest{Name: name}).Execute()
		if err == nil {
			d.logf("created account %q (id %s)", name, created.Id)
			return created, nil
		}
		if is5xx(resp) && attempt < attempts {
			d.logf("create account %s: %v — retrying (%d/%d)", name, err, attempt, attempts)
			if backoff(d.ctx, attempt) != nil {
				return nil, apiErr("create account "+name, err)
			}
			continue
		}
		return nil, apiErr("create account "+name, err)
	}
}

// ensureGroup finds a programmatic signing-key group by name or creates
// it, handing the once-returned seed to OnSeed before returning. Found
// without a custodied seed is refused by name — retrying cannot
// recover a seed nobody holds.
func (d *driver) ensureGroup(accountID, name string, scope *syncp.UserPermissionLimits, existingSeed []byte) (string, []byte, error) {
	for attempt := 1; ; attempt++ {
		groups, resp, err := d.client.AccountAPI.ListAccountSkGroup(d.ctx, accountID).Execute()
		if err != nil {
			if is5xx(resp) && attempt < attempts {
				if backoff(d.ctx, attempt) != nil {
					return "", nil, apiErr("list signing-key groups", err)
				}
				continue
			}
			return "", nil, apiErr("list signing-key groups", err)
		}
		for _, g := range groups.Items {
			if g.Name != name {
				continue
			}
			if len(existingSeed) == 0 {
				return "", nil, fmt.Errorf("synadia: signing-key group %q already exists and its seed was returned exactly once, to a run that no longer holds it — disable and then delete the group in Synadia Cloud (the platform requires the disable first) and re-run", name)
			}
			d.logf("signing-key group %q exists (id %s) — reusing its custodied seed", name, g.Id)
			return g.Id, existingSeed, nil
		}
		req := syncp.SigningKeyGroupCreateRequest{Name: name, Programmatic: true}
		if scope != nil {
			req.Scope = scope
		}
		created, resp, err := d.client.AccountAPI.CreateAccountSkGroup(d.ctx, accountID).
			SigningKeyGroupCreateRequest(req).Execute()
		if err != nil {
			if is5xx(resp) && attempt < attempts {
				d.logf("create signing-key group %s: %v — retrying (%d/%d)", name, err, attempt, attempts)
				if backoff(d.ctx, attempt) != nil {
					return "", nil, apiErr("create signing-key group "+name, err)
				}
				continue
			}
			return "", nil, apiErr("create signing-key group "+name, err)
		}
		if created.Seed == nil || *created.Seed == "" {
			return "", nil, fmt.Errorf("synadia: signing-key group %q was created but no seed came back — cannot custody it", name)
		}
		seed := []byte(*created.Seed)
		// The seed leaves this process's memory NOW, before anything
		// else can fail.
		if d.onSeed != nil {
			if err := d.onSeed(name, seed); err != nil {
				return "", nil, fmt.Errorf("synadia: persisting the seed of %q: %w", name, err)
			}
		}
		d.logf("created signing-key group %q (id %s); seed custodied", name, created.Id)
		return created.Id, seed, nil
	}
}

// ensureOnDemandGroup finds or creates a NON-programmatic group — the
// kind the platform allows NATS users under (its seed stays with
// Synadia).
func (d *driver) ensureOnDemandGroup(accountID, name string) (string, error) {
	for attempt := 1; ; attempt++ {
		groups, resp, err := d.client.AccountAPI.ListAccountSkGroup(d.ctx, accountID).Execute()
		if err != nil {
			if is5xx(resp) && attempt < attempts {
				if backoff(d.ctx, attempt) != nil {
					return "", apiErr("list signing-key groups", err)
				}
				continue
			}
			return "", apiErr("list signing-key groups", err)
		}
		for _, g := range groups.Items {
			if g.Name == name && !g.Programmatic {
				d.logf("on-demand signing-key group %q exists (id %s)", name, g.Id)
				return g.Id, nil
			}
		}
		created, resp, err := d.client.AccountAPI.CreateAccountSkGroup(d.ctx, accountID).
			SigningKeyGroupCreateRequest(syncp.SigningKeyGroupCreateRequest{Name: name, Programmatic: false}).Execute()
		if err == nil {
			d.logf("created on-demand signing-key group %q (id %s)", name, created.Id)
			return created.Id, nil
		}
		if is5xx(resp) && attempt < attempts {
			d.logf("create on-demand signing-key group %s: %v — retrying (%d/%d)", name, err, attempt, attempts)
			if backoff(d.ctx, attempt) != nil {
				return "", apiErr("create on-demand signing-key group "+name, err)
			}
			continue
		}
		return "", apiErr("create on-demand signing-key group "+name, err)
	}
}

func (d *driver) ensureUser(accountID, name, skGroupID string) (string, error) {
	for attempt := 1; ; attempt++ {
		users, resp, err := d.client.AccountAPI.ListUsers(d.ctx, accountID).Execute()
		if err != nil {
			if is5xx(resp) && attempt < attempts {
				if backoff(d.ctx, attempt) != nil {
					return "", apiErr("list nats users", err)
				}
				continue
			}
			return "", apiErr("list nats users", err)
		}
		for _, u := range users.Items {
			if u.Name == name {
				d.logf("nats user %q exists (id %s)", name, u.Id)
				return u.Id, nil
			}
		}
		created, resp, err := d.client.AccountAPI.CreateUser(d.ctx, accountID).
			NatsUserCreateRequest(syncp.NatsUserCreateRequest{Name: name, SkGroupId: skGroupID}).Execute()
		if err == nil {
			d.logf("created nats user %q (id %s)", name, created.Id)
			return created.Id, nil
		}
		if is5xx(resp) && attempt < attempts {
			d.logf("create nats user %s: %v — retrying (%d/%d)", name, err, attempt, attempts)
			if backoff(d.ctx, attempt) != nil {
				return "", apiErr("create nats user "+name, err)
			}
			continue
		}
		return "", apiErr("create nats user "+name, err)
	}
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
