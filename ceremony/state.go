package ceremony

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"

	"github.com/impire-io/soulstream-core/record"
)

// The state-directory layout (contracts/state-dir.md). Paths are relative
// to the state dir; every one of them is required for a complete realm.
const (
	fileConfig          = "config.json"
	fileSentinel        = "sentinel.creds"
	dirKeys             = "keys"
	dirUsers            = "users"
	fileOperator        = "keys/operator.nk"
	fileSysSeed         = "keys/sys.nk"
	fileSysJWT          = "keys/sys.jwt"
	fileAuthSeed        = "keys/auth.nk"
	fileAuthJWT         = "keys/auth.jwt"
	fileAuthSigning     = "keys/auth-signing.nk"
	fileRealmSeed       = "keys/realm.nk"
	fileRealmJWT        = "keys/realm.jwt"
	fileRealmSigning    = "keys/realm-signing.nk"
	fileWorkloadSigning = "keys/workload-signing.nk"
	fileAgentSigning    = "keys/agent-signing.nk"
	// AgentSigningFile is fileAgentSigning's exported name: launch
	// refusals cite it so the operator sees exactly what the realm lacks
	// (specs/013).
	AgentSigningFile   = fileAgentSigning
	fileCallout        = "keys/callout.xk"
	fileVaultFirst     = "keys/vault-first.xk"
	fileSurface        = "keys/surface.xk"
	fileServiceCreds   = "users/service.creds"
	fileIssuerCreds    = "users/issuer.creds"
	fileOpsCreds       = "users/ops.creds"
	fileArchivistCreds = "users/archivist.creds"
	fileRunnerCreds    = "users/runner.creds"
	fileSignInCreds    = "users/signin.creds"

	// BYO NATS artifacts (design 0003): the issuer user's seed (its
	// public key rides in the kit) and the kit document itself.
	fileIssuerUserSeed = "keys/issuer-user.nk"
	fileKit            = "byo-kit.md"
)

// KitPath is where the generated kit lives under dir (BYO self-hosted).
func KitPath(dir string) string { return filepath.Join(dir, fileKit) }

// ErrIncomplete marks a state directory whose keys exist but whose
// founding never finished (no sentinel): the ceremony was interrupted
// before first boot. The recovery for a never-booted directory is
// deleting it and running init fresh — trust roots are cheap before
// first boot and dangerous to half-trust after.
var ErrIncomplete = errors.New("ceremony: incomplete state directory — founding never finished; delete the directory and run init again")

// config is contracts/config.md: listen + realm fixed at founding, plus
// the per-plane blocks (design §2 — absent block or field means enabled).
type config struct {
	Listen string `json:"listen,omitempty"`
	Realm  string `json:"realm"`
	// RecordVersion is the canonical record version this realm was
	// founded under. Absent means pre-v2 (hq episode 0112's clean
	// break): refused by name with the re-founding migration, never
	// silently mixed — a v2 signature binds the realm key and names the
	// acting credential, and old history cannot be re-signed.
	RecordVersion int `json:"record_version,omitempty"`
	// BYO is design 0003 §6's block: founding on a server soulstream
	// does not run. Present ⇒ the embedded server is off and every
	// plane dials URL. Mutually exclusive with Listen, refused by name.
	BYO    *byoConfig `json:"byo,omitempty"`
	Planes struct {
		Memory planeConfig `json:"memory"`
		// MCP is the assistants' endpoint, named by function. Absent
		// means enabled — the plane is on by default.
		MCP *planeConfig `json:"mcp,omitempty"`
		// SignIn is the bundled sign-in service. A pointer on purpose:
		// the block's absence means disabled.
		SignIn *signinConfig `json:"signin,omitempty"`
		// Shell is a pointer for the same reason.
		Shell *helmConfig `json:"shell,omitempty"`
		// Dispatcher and Inference are the thinking house (specs/014),
		// both opt-in: an absent block is an absent plane, and a realm
		// that names neither runs exactly as it ran before they existed.
		Dispatcher *dispatcherConfig `json:"dispatcher,omitempty"`
		Inference  *inferenceConfig  `json:"inference,omitempty"`
		// Door and Fold are the byname-era keys. They are not read —
		// pre-v1 renames are clean breaks (design 0001 §2) — but they
		// are still *detected*, so a realm founded under them is refused
		// by name instead of silently misread.
		Door json.RawMessage `json:"door,omitempty"`
		Fold json.RawMessage `json:"fold,omitempty"`
	} `json:"planes"`
}

// byoConfig is the bring-your-own-server block. AuthAccount and
// RealmAccount are the two public keys the account half hands back —
// the only thing that crosses back, and it is public (design 0003 §4).
type byoConfig struct {
	Flavour      string         `json:"flavour"`
	URL          string         `json:"url"`
	AuthAccount  string         `json:"auth_account,omitempty"`
	RealmAccount string         `json:"realm_account,omitempty"`
	Synadia      *synadiaConfig `json:"synadia,omitempty"`
}

// synadiaConfig names the Synadia Cloud system; the API token arrives by
// environment (SOULSTREAM_SYNADIA_TOKEN) and is never persisted.
type synadiaConfig struct {
	System string `json:"system,omitempty"`
}

// signinConfig is the bundled sign-in service's block (opt-in).
type signinConfig struct {
	Enabled  *bool  `json:"enabled,omitempty"`
	Listen   string `json:"listen,omitempty"`
	Issuer   string `json:"issuer,omitempty"`
	Audience string `json:"audience,omitempty"`
}

// helmConfig is the bundled human cockpit's block (soulstream-shell). The shell
// has no issuer of its own: sessions sign in against the deployment's
// AS — the bundled sign-in service by default, or planes.mcp.auth_issuer.
// PublicURL is the origin browsers reach the console on when the
// deployment fronts the loopback listener (the shell's OAuth callback
// is built from it); absent means the bound address is the reachable
// origin.
type helmConfig struct {
	Enabled   *bool  `json:"enabled,omitempty"`
	Listen    string `json:"listen,omitempty"`
	PublicURL string `json:"public_url,omitempty"`
}

// dispatcherConfig is the standing serve arm's block (workloads design
// 0007). Placements is a topic NAME, not a path: the plane resolves it
// against the realm's board and starts the topic only in its absence, so
// the same config names the same topic across restarts. Harness and
// Template are the wrap lane's own two ways of saying which assistant
// runs — a preset name, or a template file that overrides it.
type dispatcherConfig struct {
	Enabled    *bool  `json:"enabled,omitempty"`
	Placements string `json:"placements,omitempty"`
	Harness    string `json:"harness,omitempty"`
	Template   string `json:"template,omitempty"`
}

// inferenceConfig is the plane the realm's agents think through
// (inference design 0001): the door on a loopback listener, and the
// instances this process serves.
type inferenceConfig struct {
	Enabled   *bool            `json:"enabled,omitempty"`
	Listen    string           `json:"listen,omitempty"`
	Instances []instanceConfig `json:"instances,omitempty"`
}

// instanceConfig declares one instance: one adapter, one model, one
// capability pool. Secret names the provider credential's path in the
// plane principal's D36 tree — a path, never a value; a credential in
// config.json is refused by the shape of this struct.
type instanceConfig struct {
	Adapter    string `json:"adapter"`
	Model      string `json:"model"`
	Capability string `json:"capability,omitempty"`
	Tags       string `json:"tags,omitempty"`
	Secret     string `json:"secret,omitempty"`
}

type planeConfig struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Listen  string `json:"listen,omitempty"`

	// MCP only (contracts/config.md, the public-mode addendum): the
	// advertised public address, the external AS's issuer URL, and the
	// deployment's fixed token audience. All three or none.
	PublicURL    string `json:"public_url,omitempty"`
	AuthIssuer   string `json:"auth_issuer,omitempty"`
	AuthAudience string `json:"auth_audience,omitempty"`
}

// enabled is nil-safe: an absent block is an enabled plane (design §2).
func (p *planeConfig) enabled() bool { return p == nil || p.Enabled == nil || *p.Enabled }

// secretFiles maps relative path → the State field's bytes, for Save and
// Load symmetry. JWTs and creds are secrets-adjacent and get 0600 too.
// In BYO mode the inventory is smaller by construction — no operator, no
// SYS, no account masters, no account JWTs (they live in the substrate's
// resolver) — and grows as the phases complete, so empty entries are
// skipped rather than written as empty files.
func (s *State) files() map[string][]byte {
	if s.BYO() {
		m := map[string][]byte{}
		for rel, data := range map[string][]byte{
			fileAuthSigning:     s.AuthSigningSeed,
			fileRealmSigning:    s.RealmSigningSeed,
			fileWorkloadSigning: s.WorkloadSigningSeed,
			fileAgentSigning:    s.AgentSigningSeed,
			fileCallout:         s.CalloutSeed,
			fileVaultFirst:      s.VaultFirstSeed,
			fileSurface:         s.SurfaceSeed,
			fileIssuerUserSeed:  s.IssuerUserSeed,
			fileServiceCreds:    s.ServiceCreds,
			fileIssuerCreds:     s.IssuerCreds,
			fileOpsCreds:        s.OpsCreds,
			fileArchivistCreds:  s.ArchivistCreds,
			fileRunnerCreds:     s.RunnerCreds,
			fileSignInCreds:     s.SignInCreds,
		} {
			if len(data) > 0 {
				m[rel] = data
			}
		}
		return m
	}
	m := map[string][]byte{
		fileOperator:        s.OperatorSeed,
		fileSysSeed:         s.SysSeed,
		fileSysJWT:          []byte(s.SysJWT),
		fileAuthSeed:        s.AuthSeed,
		fileAuthJWT:         []byte(s.AuthJWT),
		fileAuthSigning:     s.AuthSigningSeed,
		fileRealmSeed:       s.RealmSeed,
		fileRealmJWT:        []byte(s.RealmJWT),
		fileRealmSigning:    s.RealmSigningSeed,
		fileWorkloadSigning: s.WorkloadSigningSeed,
		fileCallout:         s.CalloutSeed,
		fileVaultFirst:      s.VaultFirstSeed,
		fileSurface:         s.SurfaceSeed,
		fileServiceCreds:    s.ServiceCreds,
		fileIssuerCreds:     s.IssuerCreds,
		fileOpsCreds:        s.OpsCreds,
		fileArchivistCreds:  s.ArchivistCreds,
		fileRunnerCreds:     s.RunnerCreds,
		fileSignInCreds:     s.SignInCreds,
	}
	// The agent capability key is founding-optional (specs/013): a state
	// loaded from a pre-capability realm has none, and re-saving it must
	// not write an empty seed file.
	if len(s.AgentSigningSeed) > 0 {
		m[fileAgentSigning] = s.AgentSigningSeed
	}
	return m
}

// Save persists the ceremony into dir: directories 0700, files 0600, and
// a refusal — not a downgrade — if the filesystem cannot hold those modes.
// The sentinel is NOT written here: it is the founding-complete marker,
// written last by WriteSentinel after the founding acts succeed.
func (s *State) Save(dir string) error {
	for _, d := range []string{dir, filepath.Join(dir, dirKeys), filepath.Join(dir, dirUsers)} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("ceremony: %w", err)
		}
		// Tighten rather than refuse: a pre-made empty dir (mkdir with a
		// permissive umask) is ours to own. The refusal below is for
		// filesystems where the chmod silently does not hold.
		if err := os.Chmod(d, 0o700); err != nil {
			return fmt.Errorf("ceremony: %w", err)
		}
		if err := requireMode(d, true); err != nil {
			return err
		}
	}
	var c config
	c.Listen = s.Listen
	c.Realm = s.Realm
	// The canonical version this realm is founded under (hq 0112): an
	// older build's realm is refused by name at load, never mixed.
	c.RecordVersion = record.Version
	if s.BYO() {
		c.BYO = &byoConfig{Flavour: s.BYOFlavour, URL: s.BYOURL,
			AuthAccount: s.AuthPub, RealmAccount: s.RealmPub}
		if s.SynadiaSystem != "" {
			c.BYO.Synadia = &synadiaConfig{System: s.SynadiaSystem}
		}
	}
	memEnabled := s.MemoryEnabled
	c.Planes.Memory.Enabled = &memEnabled
	mcpEnabled := s.MCPEnabled
	c.Planes.MCP = &planeConfig{Enabled: &mcpEnabled, Listen: s.MCPListen,
		PublicURL: s.MCPPublicURL, AuthIssuer: s.MCPAuthIssuer,
		AuthAudience: s.MCPAuthAudience}
	signinEnabled := s.SignInEnabled
	c.Planes.SignIn = &signinConfig{Enabled: &signinEnabled, Listen: s.SignInListen,
		Issuer: s.SignInIssuer, Audience: s.SignInAudience}
	helmEnabled := s.HelmEnabled
	c.Planes.Shell = &helmConfig{Enabled: &helmEnabled, Listen: s.HelmListen,
		PublicURL: s.HelmPublicURL}
	// The thinking house's two blocks are written only when they are on.
	// Founding never turns them on, and an absent block must stay absent:
	// writing `enabled: false` would make every fresh realm carry planes
	// it does not have.
	if s.DispatcherEnabled {
		on := true
		c.Planes.Dispatcher = &dispatcherConfig{Enabled: &on, Placements: s.DispatcherPlacements,
			Harness: s.DispatcherHarness, Template: s.DispatcherTemplate}
	}
	if s.InferenceEnabled {
		on := true
		c.Planes.Inference = &inferenceConfig{Enabled: &on, Listen: s.InferenceListen}
		for _, in := range s.InferenceInstances {
			c.Planes.Inference.Instances = append(c.Planes.Inference.Instances, instanceConfig(in))
		}
	}
	cfg, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("ceremony: config: %w", err)
	}
	all := s.files()
	all[fileConfig] = append(cfg, '\n')
	for rel, data := range all {
		path := filepath.Join(dir, rel)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return fmt.Errorf("ceremony: %w", err)
		}
		if err := requireMode(path, false); err != nil {
			return err
		}
	}
	return nil
}

// WriteSentinel persists the deny-all bearer creds as the LAST artifact
// of the founding run — its presence is what marks the realm founded.
func WriteSentinel(dir, creds string) error {
	path := filepath.Join(dir, fileSentinel)
	if err := os.WriteFile(path, []byte(creds), 0o600); err != nil {
		return fmt.Errorf("ceremony: sentinel: %w", err)
	}
	return requireMode(path, false)
}

// SentinelPath returns where the sentinel creds live under dir.
func SentinelPath(dir string) string { return filepath.Join(dir, fileSentinel) }

// UserCredsPath returns the path of a bypass-lane creds file ("service",
// "issuer", "ops") under dir.
func UserCredsPath(dir, name string) string {
	return filepath.Join(dir, dirUsers, name+".creds")
}

// Founded reports whether the founding completed: the sentinel exists.
func Founded(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, fileSentinel))
	return err == nil && !info.IsDir()
}

// Empty reports whether dir is absent or an empty directory — the fresh
// state init founds into.
func Empty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("ceremony: %w", err)
	}
	return len(entries) == 0, nil
}

// Load reads and parses a persisted ceremony. It checks shape (files
// exist, seeds decode, JWTs parse); Verify adds the cross-checks.
func Load(dir string) (*State, error) {
	s := &State{}
	read := func(rel string) ([]byte, error) {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return nil, fmt.Errorf("ceremony: missing or unreadable %s: %w", rel, err)
		}
		return data, nil
	}
	cfgData, err := read(fileConfig)
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		return nil, fmt.Errorf("ceremony: damaged %s: %w", fileConfig, err)
	}
	s.Listen = cfg.Listen
	s.Realm = cfg.Realm
	// The soulstream *client* tooling historically keeps its own
	// config.json (context/persona, ed25519 persona keys) under the
	// same default path this state dir resolves to. A realm must never
	// squat on it, and reading it as a ceremony is a named refusal —
	// not a missing-file error (first hit: a machine that used the
	// component CLI before the product, 2026-08-16).
	if cfg.Listen == "" && cfg.BYO == nil {
		var clientCfg struct {
			Context string `json:"context"`
			Persona string `json:"persona"`
		}
		if json.Unmarshal(cfgData, &clientCfg) == nil &&
			(clientCfg.Context != "" || clientCfg.Persona != "") {
			return nil, fmt.Errorf("ceremony: this directory holds the soulstream client's configuration (context %q, persona %q), not a realm — the realm needs its own directory: pass --state DIR or set SOULSTREAM_STATE", clientCfg.Context, clientCfg.Persona)
		}
	}
	if cfg.RecordVersion != record.Version {
		return nil, fmt.Errorf("ceremony: this realm was founded under canonical record v%d and this build writes v%d — pre-v1 breaks are clean (hq episode 0112: signatures now bind the realm key, and every record names its acting credential). Existing signed history cannot be re-signed; re-found a fresh realm (`soulstream init --state DIR`) and keep the old directory for reading with the matching older build", cfg.RecordVersion, record.Version)
	}
	if cfg.Planes.Door != nil || cfg.Planes.Fold != nil {
		return nil, fmt.Errorf("ceremony: %s carries byname-era plane keys — pre-v1 renames are clean breaks (design 0001 §2). Migrate by hand: rename the keys (door→mcp, fold→signin), `mv users/fold.creds users/signin.creds`, `mv fold/ signin/` — or re-init a fresh realm", fileConfig)
	}
	s.MemoryEnabled = cfg.Planes.Memory.enabled()
	mcpBlock := cfg.Planes.MCP
	s.MCPEnabled = mcpBlock.enabled()
	if mcpBlock != nil {
		s.MCPListen = mcpBlock.Listen
		s.MCPPublicURL = mcpBlock.PublicURL
		s.MCPAuthIssuer = mcpBlock.AuthIssuer
		s.MCPAuthAudience = mcpBlock.AuthAudience
	}
	if s.MCPListen == "" {
		s.MCPListen = "127.0.0.1:8080"
	}
	if f := cfg.Planes.SignIn; f != nil && (f.Enabled == nil || *f.Enabled) {
		s.SignInEnabled = true
		s.SignInListen = f.Listen
		if s.SignInListen == "" {
			s.SignInListen = "127.0.0.1:8378"
		}
		s.SignInIssuer = f.Issuer
		if s.SignInIssuer == "" {
			// The issuer host is WebAuthn's RP ID, and a browser refuses
			// a bare IP there — so the local default is localhost (a
			// WebAuthn secure-context name that still resolves to the
			// loopback bind), never 127.0.0.1. A public deployment sets
			// planes.signin.issuer to its fronted name.
			_, port, _ := net.SplitHostPort(s.SignInListen)
			s.SignInIssuer = "http://localhost:" + port
		}
		s.SignInAudience = f.Audience
		if s.SignInAudience == "" {
			s.SignInAudience = "soulstream-" + s.Realm
		}
		// The default wiring (soulstream-idp M5's distribution story): public
		// door mode with no AS named points at the bundled fold. In
		// public mode planes.signin.issuer must be the fronted sign-in URL,
		// so the browser the MCP endpoint sends a user to is actually reachable.
		if s.MCPPublicURL != "" && s.MCPAuthIssuer == "" {
			s.MCPAuthIssuer = s.SignInIssuer
			s.MCPAuthAudience = s.SignInAudience
		}
	}
	if h := cfg.Planes.Shell; h != nil && (h.Enabled == nil || *h.Enabled) {
		s.HelmEnabled = true
		s.HelmListen = h.Listen
		if s.HelmListen == "" {
			s.HelmListen = "127.0.0.1:8500"
		}
		s.HelmPublicURL = h.PublicURL
	}
	if d := cfg.Planes.Dispatcher; d != nil && (d.Enabled == nil || *d.Enabled) {
		s.DispatcherEnabled = true
		s.DispatcherPlacements = d.Placements
		if s.DispatcherPlacements == "" {
			s.DispatcherPlacements = DefaultPlacements
		}
		s.DispatcherHarness = d.Harness
		s.DispatcherTemplate = d.Template
	}
	if i := cfg.Planes.Inference; i != nil && (i.Enabled == nil || *i.Enabled) {
		s.InferenceEnabled = true
		s.InferenceListen = i.Listen
		if s.InferenceListen == "" {
			s.InferenceListen = "127.0.0.1:8600"
		}
		for _, in := range i.Instances {
			capability := in.Capability
			if capability == "" {
				capability = "chat"
			}
			s.InferenceInstances = append(s.InferenceInstances, InferenceInstance{
				Adapter: in.Adapter, Model: in.Model, Capability: capability,
				Tags: in.Tags, Secret: in.Secret})
		}
	}

	// BYO mode loads its own, smaller inventory: signing keys and curve
	// keys always; the issuer-user and callout seeds when the flavour
	// generated them; creds only once minted — required the moment the
	// realm is founded, optional in the awaiting states before it.
	if cfg.BYO != nil {
		s.BYOFlavour = cfg.BYO.Flavour
		s.BYOURL = cfg.BYO.URL
		s.AuthPub = cfg.BYO.AuthAccount
		s.RealmPub = cfg.BYO.RealmAccount
		if cfg.BYO.Synadia != nil {
			s.SynadiaSystem = cfg.BYO.Synadia.System
		}
		for _, sf := range []struct {
			rel   string
			dst   *[]byte
			check func([]byte) error
		}{
			{fileVaultFirst, &s.VaultFirstSeed, kind(nkeys.PrefixByteCurve)},
			{fileSurface, &s.SurfaceSeed, kind(nkeys.PrefixByteCurve)},
		} {
			data, err := read(sf.rel)
			if err != nil {
				return nil, err
			}
			if err := sf.check(data); err != nil {
				return nil, fmt.Errorf("ceremony: damaged %s: %w", sf.rel, err)
			}
			*sf.dst = data
		}
		// The signing keys are load-if-present: self-hosted generates
		// them at phase 1 (Verify requires them there), but on Synadia
		// Cloud they exist only after the platform returns them — the
		// awaiting-driver state has none, and a resume after a platform
		// timeout must load (found live, 2026-08-16: the first BYON
		// founding hit a 500 mid-account-half and could not resume).
		for _, of := range []struct {
			rel   string
			dst   *[]byte
			check func([]byte) error
		}{
			{fileAuthSigning, &s.AuthSigningSeed, kind(nkeys.PrefixByteAccount)},
			{fileRealmSigning, &s.RealmSigningSeed, kind(nkeys.PrefixByteAccount)},
			{fileWorkloadSigning, &s.WorkloadSigningSeed, kind(nkeys.PrefixByteAccount)},
			{fileAgentSigning, &s.AgentSigningSeed, kind(nkeys.PrefixByteAccount)},
			{fileCallout, &s.CalloutSeed, kind(nkeys.PrefixByteCurve)},
			{fileIssuerUserSeed, &s.IssuerUserSeed, kind(nkeys.PrefixByteUser)},
		} {
			data, err := os.ReadFile(filepath.Join(dir, of.rel))
			if err != nil {
				continue
			}
			if err := of.check(data); err != nil {
				return nil, fmt.Errorf("ceremony: damaged %s: %w", of.rel, err)
			}
			*of.dst = data
		}
		for _, p := range []struct {
			rel  string
			seed []byte
			dst  *string
		}{
			{fileAuthSigning, s.AuthSigningSeed, &s.AuthSigningPub},
			{fileRealmSigning, s.RealmSigningSeed, &s.RealmSigningPub},
			{fileWorkloadSigning, s.WorkloadSigningSeed, &s.WorkloadSigningPub},
			{fileAgentSigning, s.AgentSigningSeed, &s.AgentSigningPub},
			{fileCallout, s.CalloutSeed, &s.CalloutPub},
			{fileIssuerUserSeed, s.IssuerUserSeed, &s.IssuerUserPub},
		} {
			if len(p.seed) == 0 {
				continue
			}
			if *p.dst, err = pubOf(p.seed); err != nil {
				return nil, fmt.Errorf("ceremony: %s: %w", p.rel, err)
			}
		}
		founded := Founded(dir)
		if founded {
			for rel, seed := range map[string][]byte{
				fileAuthSigning:     s.AuthSigningSeed,
				fileRealmSigning:    s.RealmSigningSeed,
				fileWorkloadSigning: s.WorkloadSigningSeed,
			} {
				if len(seed) == 0 {
					return nil, fmt.Errorf("ceremony: missing or unreadable %s on a founded realm", rel)
				}
			}
		}
		for _, cf := range []struct {
			rel string
			dst *[]byte
		}{{fileServiceCreds, &s.ServiceCreds}, {fileIssuerCreds, &s.IssuerCreds},
			{fileOpsCreds, &s.OpsCreds}, {fileArchivistCreds, &s.ArchivistCreds},
			{fileRunnerCreds, &s.RunnerCreds}, {fileSignInCreds, &s.SignInCreds}} {
			data, err := os.ReadFile(filepath.Join(dir, cf.rel))
			if err != nil {
				if founded {
					return nil, fmt.Errorf("ceremony: missing or unreadable %s: %w", cf.rel, err)
				}
				continue
			}
			if _, err := jwt.ParseDecoratedJWT(data); err != nil {
				return nil, fmt.Errorf("ceremony: damaged %s: %w", cf.rel, err)
			}
			*cf.dst = data
		}
		return s, nil
	}

	seeds := []struct {
		rel   string
		dst   *[]byte
		check func([]byte) error
	}{
		{fileOperator, &s.OperatorSeed, kind(nkeys.PrefixByteOperator)},
		{fileSysSeed, &s.SysSeed, kind(nkeys.PrefixByteAccount)},
		{fileAuthSeed, &s.AuthSeed, kind(nkeys.PrefixByteAccount)},
		{fileAuthSigning, &s.AuthSigningSeed, kind(nkeys.PrefixByteAccount)},
		{fileRealmSeed, &s.RealmSeed, kind(nkeys.PrefixByteAccount)},
		{fileRealmSigning, &s.RealmSigningSeed, kind(nkeys.PrefixByteAccount)},
		{fileWorkloadSigning, &s.WorkloadSigningSeed, kind(nkeys.PrefixByteAccount)},
		{fileCallout, &s.CalloutSeed, kind(nkeys.PrefixByteCurve)},
		{fileVaultFirst, &s.VaultFirstSeed, kind(nkeys.PrefixByteCurve)},
		{fileSurface, &s.SurfaceSeed, kind(nkeys.PrefixByteCurve)},
	}
	for _, sf := range seeds {
		data, err := read(sf.rel)
		if err != nil {
			return nil, err
		}
		if err := sf.check(data); err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: %w", sf.rel, err)
		}
		*sf.dst = data
	}
	if s.OperatorPub, err = pubOf(s.OperatorSeed); err != nil {
		return nil, fmt.Errorf("ceremony: %s: %w", fileOperator, err)
	}
	if s.SysPub, err = pubOf(s.SysSeed); err != nil {
		return nil, fmt.Errorf("ceremony: %s: %w", fileSysSeed, err)
	}
	if s.AuthPub, err = pubOf(s.AuthSeed); err != nil {
		return nil, fmt.Errorf("ceremony: %s: %w", fileAuthSeed, err)
	}
	if s.AuthSigningPub, err = pubOf(s.AuthSigningSeed); err != nil {
		return nil, fmt.Errorf("ceremony: %s: %w", fileAuthSigning, err)
	}
	if s.RealmPub, err = pubOf(s.RealmSeed); err != nil {
		return nil, fmt.Errorf("ceremony: %s: %w", fileRealmSeed, err)
	}
	if s.RealmSigningPub, err = pubOf(s.RealmSigningSeed); err != nil {
		return nil, fmt.Errorf("ceremony: %s: %w", fileRealmSigning, err)
	}
	if s.WorkloadSigningPub, err = pubOf(s.WorkloadSigningSeed); err != nil {
		return nil, fmt.Errorf("ceremony: %s: %w", fileWorkloadSigning, err)
	}
	// The agent capability key is load-if-present (specs/013): realms
	// founded before capability-minting lack it and stay valid — a
	// capability-bearing declaration there refuses by name at launch.
	if data, err := os.ReadFile(filepath.Join(dir, fileAgentSigning)); err == nil {
		if err := kind(nkeys.PrefixByteAccount)(data); err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: %w", fileAgentSigning, err)
		}
		s.AgentSigningSeed = data
		if s.AgentSigningPub, err = pubOf(data); err != nil {
			return nil, fmt.Errorf("ceremony: %s: %w", fileAgentSigning, err)
		}
	}
	if s.CalloutPub, err = pubOf(s.CalloutSeed); err != nil {
		return nil, fmt.Errorf("ceremony: %s: %w", fileCallout, err)
	}

	for _, jf := range []struct {
		rel string
		dst *string
	}{{fileSysJWT, &s.SysJWT}, {fileAuthJWT, &s.AuthJWT}, {fileRealmJWT, &s.RealmJWT}} {
		data, err := read(jf.rel)
		if err != nil {
			return nil, err
		}
		if _, err := jwt.DecodeAccountClaims(string(data)); err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: %w", jf.rel, err)
		}
		*jf.dst = string(data)
	}

	for _, cf := range []struct {
		rel string
		dst *[]byte
	}{{fileServiceCreds, &s.ServiceCreds}, {fileIssuerCreds, &s.IssuerCreds},
		{fileOpsCreds, &s.OpsCreds}, {fileArchivistCreds, &s.ArchivistCreds},
		{fileRunnerCreds, &s.RunnerCreds}} {
		data, err := read(cf.rel)
		if err != nil {
			return nil, err
		}
		if _, err := jwt.ParseDecoratedJWT(data); err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: %w", cf.rel, err)
		}
		*cf.dst = data
	}
	// The sign-in plane's creds are required only when it is enabled: a
	// state dir founded before the plane existed loads fine with it off,
	// and enabling it there is a named refusal.
	data, err := os.ReadFile(filepath.Join(dir, fileSignInCreds))
	if err == nil {
		if _, err := jwt.ParseDecoratedJWT(data); err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: %w", fileSignInCreds, err)
		}
		s.SignInCreds = data
	} else if s.SignInEnabled {
		return nil, fmt.Errorf("ceremony: planes.signin is enabled but %s is missing — this realm was founded before the sign-in plane existed; disable the plane or re-init a fresh realm", fileSignInCreds)
	}
	return s, nil
}

// Verify is Load plus the cross-checks of data-model.md: JWT subjects
// match their seeds, the AUTH JWT carries the admission machinery wired
// to the persisted callout key, the realm JWT carries the scoped signing
// key, and the listener is loopback. The first failure is named.
func Verify(dir string) (*State, error) {
	s, err := Load(dir)
	if err != nil {
		return nil, err
	}
	if s.BYO() {
		if err := s.verifyBYO(); err != nil {
			return nil, err
		}
	} else {
		if err := s.verifyEmbedded(); err != nil {
			return nil, err
		}
	}
	if s.Realm == "" {
		return nil, fmt.Errorf("ceremony: damaged %s: realm name missing", fileConfig)
	}
	if s.MCPEnabled {
		dhost, _, err := net.SplitHostPort(s.MCPListen)
		if err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: door listen %q: %w", fileConfig, s.MCPListen, err)
		}
		if ip := net.ParseIP(dhost); dhost != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return nil, fmt.Errorf("ceremony: %s door listen %q is not loopback", fileConfig, s.MCPListen)
		}
	}
	// Public mode is a package deal: the MCP endpoint needs the advertised URL
	// and the AS; the identity plane's OIDC lane needs the AS and the
	// audience. A partial declaration is a config error, never a
	// silently absent OAuth story.
	publicFields := 0
	for _, f := range []string{s.MCPPublicURL, s.MCPAuthIssuer, s.MCPAuthAudience} {
		if f != "" {
			publicFields++
		}
	}
	if publicFields != 0 && publicFields != 3 {
		return nil, fmt.Errorf("ceremony: %s public MCP mode needs all of planes.mcp.public_url, auth_issuer, auth_audience (or none)", fileConfig)
	}
	if publicFields == 3 && !s.MCPEnabled {
		return nil, fmt.Errorf("ceremony: %s declares public MCP mode with the MCP plane disabled", fileConfig)
	}
	if s.SignInEnabled {
		fhost, _, err := net.SplitHostPort(s.SignInListen)
		if err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: fold listen %q: %w", fileConfig, s.SignInListen, err)
		}
		if ip := net.ParseIP(fhost); fhost != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return nil, fmt.Errorf("ceremony: %s fold listen %q is not loopback", fileConfig, s.SignInListen)
		}
		// The sign-in service and the MCP endpoint are distinct services on distinct
		// listeners; a shared address would have them fight for the same
		// port (and, fronted, the same public route). Refuse it by name —
		// except the ephemeral :0, which resolves to different real ports.
		if _, fport, err := net.SplitHostPort(s.SignInListen); err == nil && fport != "0" &&
			s.MCPEnabled && s.SignInListen == s.MCPListen {
			return nil, fmt.Errorf("ceremony: %s planes.signin.listen and planes.mcp.listen are both %q — they are separate services and need separate addresses", fileConfig, s.SignInListen)
		}
		// The issuer host becomes WebAuthn's RP ID; a browser refuses a
		// bare IP there. Catch the footgun at load, not at first
		// enrolment.
		iss, err := url.Parse(s.SignInIssuer)
		if err != nil || iss.Hostname() == "" {
			return nil, fmt.Errorf("ceremony: %s planes.signin.issuer %q is not a URL", fileConfig, s.SignInIssuer)
		}
		if h := iss.Hostname(); h != "localhost" && net.ParseIP(h) != nil {
			return nil, fmt.Errorf("ceremony: %s planes.signin.issuer host %q is a bare IP — WebAuthn refuses it as a relying-party id; use localhost or a real hostname", fileConfig, h)
		}
	}
	if s.HelmEnabled {
		hhost, hport, err := net.SplitHostPort(s.HelmListen)
		if err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: shell listen %q: %w", fileConfig, s.HelmListen, err)
		}
		if ip := net.ParseIP(hhost); hhost != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return nil, fmt.Errorf("ceremony: %s shell listen %q is not loopback", fileConfig, s.HelmListen)
		}
		if hport != "0" {
			if s.MCPEnabled && s.HelmListen == s.MCPListen {
				return nil, fmt.Errorf("ceremony: %s planes.shell.listen and planes.mcp.listen are both %q — they are separate services and need separate addresses", fileConfig, s.HelmListen)
			}
			if s.SignInEnabled && s.HelmListen == s.SignInListen {
				return nil, fmt.Errorf("ceremony: %s planes.shell.listen and planes.signin.listen are both %q — they are separate services and need separate addresses", fileConfig, s.HelmListen)
			}
		}
		if s.HelmPublicURL != "" {
			pu, err := url.Parse(s.HelmPublicURL)
			if err != nil || pu.Scheme == "" || pu.Host == "" {
				return nil, fmt.Errorf("ceremony: %s planes.shell.public_url %q is not a URL", fileConfig, s.HelmPublicURL)
			}
		}
		// Sessions need a sign-in issuer: the bundled fold, or an
		// external AS via planes.mcp.auth_issuer.
		if s.MCPAuthIssuer == "" && !s.SignInEnabled {
			return nil, fmt.Errorf("ceremony: %s enables the shell plane with no sign-in issuer — enable planes.signin or set planes.mcp.auth_issuer", fileConfig)
		}
	}
	if err := s.verifyThinking(); err != nil {
		return nil, err
	}
	return s, nil
}

// verifyThinking is the thinking house's configuration cross-checks
// (specs/014): a dispatcher that names no assistant would refuse every
// placement it won, and an instance the house cannot construct would
// fail at the first request instead of at startup. Both are refused here,
// before anything runs.
func (s *State) verifyThinking() error {
	if s.DispatcherEnabled && s.DispatcherHarness == "" && s.DispatcherTemplate == "" {
		return fmt.Errorf("ceremony: %s enables the dispatcher plane but names no assistant — set planes.dispatcher.harness (a preset) or planes.dispatcher.template (a template file)", fileConfig)
	}
	if !s.InferenceEnabled {
		return nil
	}
	host, port, err := net.SplitHostPort(s.InferenceListen)
	if err != nil {
		return fmt.Errorf("ceremony: damaged %s: inference listen %q: %w", fileConfig, s.InferenceListen, err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("ceremony: %s inference listen %q is not loopback", fileConfig, s.InferenceListen)
	}
	if port != "0" {
		for _, other := range []struct{ name, listen string }{
			{"planes.mcp.listen", listenIf(s.MCPEnabled, s.MCPListen)},
			{"planes.signin.listen", listenIf(s.SignInEnabled, s.SignInListen)},
			{"planes.shell.listen", listenIf(s.HelmEnabled, s.HelmListen)},
		} {
			if other.listen == s.InferenceListen {
				return fmt.Errorf("ceremony: %s planes.inference.listen and %s are both %q — they are separate services and need separate addresses", fileConfig, other.name, s.InferenceListen)
			}
		}
	}
	models := map[string]bool{}
	for i, in := range s.InferenceInstances {
		where := fmt.Sprintf("planes.inference.instances[%d]", i)
		if in.Model == "" {
			return fmt.Errorf("ceremony: %s %s names no model — an instance wraps exactly one", fileConfig, where)
		}
		if models[in.Model] {
			return fmt.Errorf("ceremony: %s declares model %q twice — a pinned name would resolve to whichever instance answered first; give each instance its own model", fileConfig, in.Model)
		}
		models[in.Model] = true
		switch in.Adapter {
		case AdapterStandin:
			if in.Secret != "" {
				return fmt.Errorf("ceremony: %s %s is the stand-in adapter and holds no provider credential — remove its secret", fileConfig, where)
			}
		case AdapterAnthropic:
			if in.Secret == "" {
				return fmt.Errorf("ceremony: %s %s needs a secret — the path in this plane's own store where its provider key rests (`soulstream provider set anthropic` writes it)", fileConfig, where)
			}
		case "":
			return fmt.Errorf("ceremony: %s %s names no adapter — the house wires %q and %q", fileConfig, where, AdapterStandin, AdapterAnthropic)
		default:
			return fmt.Errorf("ceremony: %s %s adapter %q is not one the house wires (%q, %q)", fileConfig, where, in.Adapter, AdapterStandin, AdapterAnthropic)
		}
	}
	return nil
}

// listenIf yields a plane's listener only when that plane is on — a
// disabled plane's address is not a collision.
func listenIf(enabled bool, listen string) string {
	if !enabled {
		return ""
	}
	return listen
}

// verifyEmbedded is the embedded server's cross-checks: JWT subjects
// match their seeds, the AUTH JWT carries the admission machinery wired
// to the persisted callout key, the realm JWT carries the scoped signing
// key, and the listener is loopback.
func (s *State) verifyEmbedded() error {
	sys, err := jwt.DecodeAccountClaims(s.SysJWT)
	if err != nil || sys.Subject != s.SysPub {
		return fmt.Errorf("ceremony: %s does not match %s (subject %s, key %s)", fileSysJWT, fileSysSeed, subjectOf(sys, err), s.SysPub)
	}
	auth, err := jwt.DecodeAccountClaims(s.AuthJWT)
	if err != nil || auth.Subject != s.AuthPub {
		return fmt.Errorf("ceremony: %s does not match %s", fileAuthJWT, fileAuthSeed)
	}
	if len(auth.Authorization.AuthUsers) == 0 {
		return fmt.Errorf("ceremony: %s lacks external authorization", fileAuthJWT)
	}
	if auth.Authorization.XKey != s.CalloutPub {
		return fmt.Errorf("ceremony: %s callout xkey does not match %s", fileAuthJWT, fileCallout)
	}
	if !auth.SigningKeys.Contains(s.AuthSigningPub) {
		return fmt.Errorf("ceremony: %s does not endorse %s", fileAuthJWT, fileAuthSigning)
	}
	realm, err := jwt.DecodeAccountClaims(s.RealmJWT)
	if err != nil || realm.Subject != s.RealmPub {
		return fmt.Errorf("ceremony: %s does not match %s", fileRealmJWT, fileRealmSeed)
	}
	if scope, ok := realm.SigningKeys.GetScope(s.RealmSigningPub); !ok || scope == nil {
		return fmt.Errorf("ceremony: %s lacks the scoped signing key %s", fileRealmJWT, fileRealmSigning)
	}
	// The workload minting key must be endorsed PLAIN (nil scope): a
	// scoped key would reject the minter's carried permissions (R1).
	if scope, ok := realm.SigningKeys.GetScope(s.WorkloadSigningPub); !ok || scope != nil {
		return fmt.Errorf("ceremony: %s does not endorse %s as a plain signing key", fileRealmJWT, fileWorkloadSigning)
	}
	// The agent capability key, when present, must be endorsed SCOPED:
	// its template is the entire policy for capability-minted users
	// (specs/013).
	if len(s.AgentSigningSeed) > 0 {
		if scope, ok := realm.SigningKeys.GetScope(s.AgentSigningPub); !ok || scope == nil {
			return fmt.Errorf("ceremony: %s does not endorse %s as a scoped signing key", fileRealmJWT, fileAgentSigning)
		}
	}
	host, _, err := net.SplitHostPort(s.Listen)
	if err != nil {
		return fmt.Errorf("ceremony: damaged %s: listen %q: %w", fileConfig, s.Listen, err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("ceremony: %s listen %q is not loopback", fileConfig, s.Listen)
	}
	return nil
}

// verifyBYO is the bring-your-own-server cross-checks (design 0003 §6):
// the flavour is one of the two, the substrate URL parses, the embedded
// listener is absent, and the handed-back account keys — when present —
// are two distinct account public keys. What the substrate itself
// carries is verified behaviourally at founding (node.ProbeSubstrate),
// never from here: this state holds no account JWTs to check.
func (s *State) verifyBYO() error {
	switch s.BYOFlavour {
	case FlavourSelfHosted, FlavourSynadiaCloud:
	default:
		return fmt.Errorf("ceremony: %s byo.flavour %q is not a flavour — the two flavours are %q and %q (design 0003 §1)",
			fileConfig, s.BYOFlavour, FlavourSelfHosted, FlavourSynadiaCloud)
	}
	if s.Listen != "" {
		return fmt.Errorf("ceremony: %s carries both listen and byo — the embedded listener and a bring-your-own server are mutually exclusive; remove one", fileConfig)
	}
	u, err := url.Parse(s.BYOURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("ceremony: %s byo.url %q is not a server URL (nats://host:port)", fileConfig, s.BYOURL)
	}
	if s.BYOFlavour == FlavourSelfHosted {
		for rel, seed := range map[string][]byte{
			fileIssuerUserSeed:  s.IssuerUserSeed,
			fileCallout:         s.CalloutSeed,
			fileAuthSigning:     s.AuthSigningSeed,
			fileRealmSigning:    s.RealmSigningSeed,
			fileWorkloadSigning: s.WorkloadSigningSeed,
		} {
			if len(seed) == 0 {
				return fmt.Errorf("ceremony: missing %s — self-hosted phase-1 material; re-init a fresh state dir", rel)
			}
		}
	}
	if s.BYOFlavour == FlavourSynadiaCloud && s.SynadiaSystem == "" {
		return fmt.Errorf("ceremony: %s byo.synadia.system is required for the synadia-cloud flavour", fileConfig)
	}
	for _, k := range []struct{ name, pub string }{
		{"byo.auth_account", s.AuthPub}, {"byo.realm_account", s.RealmPub},
	} {
		if k.pub != "" && !nkeys.IsValidPublicAccountKey(k.pub) {
			return fmt.Errorf("ceremony: %s %s %q is not an account public key", fileConfig, k.name, k.pub)
		}
	}
	if s.AuthPub != "" && s.AuthPub == s.RealmPub {
		return fmt.Errorf("ceremony: %s byo.auth_account and byo.realm_account are the same key — the AUTH account and the realm account are two accounts (design 0003 §2)", fileConfig)
	}
	return nil
}

// ArtifactCount is the number of persisted founding artifacts a complete
// state directory carries (config + keys + users + sentinel) — the number
// init's verify report cites. The BYO inventory is smaller by
// construction: no operator, SYS, or account master material exists on
// this side of the boundary.
func (s *State) ArtifactCount() int { return len(s.files()) + 1 + 1 } // + config + sentinel

func requireMode(path string, isDir bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("ceremony: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		kind := "file"
		if isDir {
			kind = "directory"
		}
		return fmt.Errorf("ceremony: %s %s has mode %o — this filesystem cannot hold owner-only secrets; refusing", kind, path, info.Mode().Perm())
	}
	return nil
}

func kind(prefix nkeys.PrefixByte) func([]byte) error {
	return func(seed []byte) error {
		p, _, err := nkeys.DecodeSeed(seed)
		if err != nil {
			return err
		}
		if p != prefix {
			return fmt.Errorf("seed is %v, want %v", p, prefix)
		}
		return nil
	}
}

func pubOf(seed []byte) (string, error) {
	kp, err := nkeys.FromSeed(seed)
	if err != nil {
		return "", err
	}
	return kp.PublicKey()
}

func subjectOf(c *jwt.AccountClaims, err error) string {
	if err != nil || c == nil {
		return "<unparseable>"
	}
	return c.Subject
}

// RecordVersion is the canonical record version this build writes —
// what a state directory must declare to be adoptable (hq 0112).
func RecordVersion() int { return record.Version }

// PreV2 reports whether a state directory predates the canonical-form
// break: a config that declares no version, or an older one. A
// directory this build already wrote answers false.
func PreV2(dir string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, fileConfig))
	if err != nil {
		return false, fmt.Errorf("ceremony: %w", err)
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false, fmt.Errorf("ceremony: damaged %s: %w", fileConfig, err)
	}
	return cfg.RecordVersion != record.Version, nil
}

// AdoptV2 stamps the canonical version onto a pre-break state
// directory, leaving every other field exactly as founding wrote it.
// It is the ONLY writer of that field outside founding, and it decides
// nothing: `soulstream adopt` weighs whether adoption is safe.
func AdoptV2(dir string) error {
	path := filepath.Join(dir, fileConfig)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ceremony: %w", err)
	}
	// Round-trip through a generic map so nothing this build does not
	// know about is dropped from a directory it did not write.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ceremony: damaged %s: %w", fileConfig, err)
	}
	raw["record_version"] = record.Version
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("ceremony: %w", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o600); err != nil {
		return fmt.Errorf("ceremony: %w", err)
	}
	return nil
}

// OpsConnection is how a maintenance act reaches the realm's own
// server: the URL the realm runs on (the BYO server's, or the
// configured loopback listener) and the ops credential founding left
// behind. The creds path is empty where the realm needs none.
func OpsConnection(dir string) (url, creds string, err error) {
	data, rerr := os.ReadFile(filepath.Join(dir, fileConfig))
	if rerr != nil {
		return "", "", fmt.Errorf("ceremony: %w", rerr)
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", fmt.Errorf("ceremony: damaged %s: %w", fileConfig, err)
	}
	switch {
	case cfg.BYO != nil && cfg.BYO.URL != "":
		url = cfg.BYO.URL
	case cfg.Listen != "":
		url = "nats://" + cfg.Listen
	default:
		return "", "", errors.New("ceremony: the config names neither a BYO url nor a listener")
	}
	if p := UserCredsPath(dir, "ops"); fileExists(p) {
		creds = p
	}
	return url, creds, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
