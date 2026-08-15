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
	fileCallout         = "keys/callout.xk"
	fileVaultFirst      = "keys/vault-first.xk"
	fileSurface         = "keys/surface.xk"
	fileServiceCreds    = "users/service.creds"
	fileIssuerCreds     = "users/issuer.creds"
	fileOpsCreds        = "users/ops.creds"
	fileArchivistCreds  = "users/archivist.creds"
	fileRunnerCreds     = "users/runner.creds"
	fileSignInCreds     = "users/signin.creds"
	// fileLegacySignInCreds is the byname-era spelling, read forever: a
	// founded realm's artifacts are never rewritten (design 0001 §2).
	fileLegacySignInCreds = "users/fold.creds"
)

// ErrIncomplete marks a state directory whose keys exist but whose
// founding never finished (no sentinel): the ceremony was interrupted
// before first boot. The recovery for a never-booted directory is
// deleting it and running init fresh — trust roots are cheap before
// first boot and dangerous to half-trust after.
var ErrIncomplete = errors.New("ceremony: incomplete state directory — founding never finished; delete the directory and run init again")

// config is contracts/config.md: listen + realm fixed at founding, plus
// the per-plane blocks (design §2 — absent block or field means enabled).
type config struct {
	Listen string `json:"listen"`
	Realm  string `json:"realm"`
	Planes struct {
		Memory planeConfig `json:"memory"`
		// MCP is the assistants' endpoint, named by function; Door is the
		// same block under its byname-era key, read forever so a founded
		// realm's config never stops working (design 0001 §2). Both
		// absent means enabled — the plane is on by default.
		MCP  *planeConfig `json:"mcp,omitempty"`
		Door *planeConfig `json:"door,omitempty"`
		// SignIn is the bundled sign-in service, named by function; Fold
		// is its byname-era key, read forever. Pointers on purpose: the
		// block's absence means disabled — state dirs founded before the
		// plane existed must not sprout one on upgrade.
		SignIn *signinConfig `json:"signin,omitempty"`
		Fold   *signinConfig `json:"fold,omitempty"`
		// Shell is a pointer for the same reason: state dirs founded
		// before the shell plane existed must not sprout one on upgrade.
		Shell *helmConfig `json:"shell,omitempty"`
	} `json:"planes"`
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
type helmConfig struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Listen  string `json:"listen,omitempty"`
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
func (s *State) files() map[string][]byte {
	return map[string][]byte{
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
	c.Planes.Shell = &helmConfig{Enabled: &helmEnabled, Listen: s.HelmListen}
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
	s.MemoryEnabled = cfg.Planes.Memory.enabled()
	mcpBlock := cfg.Planes.MCP
	if mcpBlock == nil {
		mcpBlock = cfg.Planes.Door
	}
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
	signinBlock := cfg.Planes.SignIn
	if signinBlock == nil {
		signinBlock = cfg.Planes.Fold
	}
	if f := signinBlock; f != nil && (f.Enabled == nil || *f.Enabled) {
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
	// and enabling it there is a named refusal. The legacy filename is
	// read forever — a founded realm's artifacts are never rewritten.
	signinRel := fileSignInCreds
	data, err := os.ReadFile(filepath.Join(dir, signinRel))
	if err != nil {
		signinRel = fileLegacySignInCreds
		data, err = os.ReadFile(filepath.Join(dir, signinRel))
	}
	if err == nil {
		if _, err := jwt.ParseDecoratedJWT(data); err != nil {
			return nil, fmt.Errorf("ceremony: damaged %s: %w", signinRel, err)
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
	sys, err := jwt.DecodeAccountClaims(s.SysJWT)
	if err != nil || sys.Subject != s.SysPub {
		return nil, fmt.Errorf("ceremony: %s does not match %s (subject %s, key %s)", fileSysJWT, fileSysSeed, subjectOf(sys, err), s.SysPub)
	}
	auth, err := jwt.DecodeAccountClaims(s.AuthJWT)
	if err != nil || auth.Subject != s.AuthPub {
		return nil, fmt.Errorf("ceremony: %s does not match %s", fileAuthJWT, fileAuthSeed)
	}
	if len(auth.Authorization.AuthUsers) == 0 {
		return nil, fmt.Errorf("ceremony: %s lacks external authorization", fileAuthJWT)
	}
	if auth.Authorization.XKey != s.CalloutPub {
		return nil, fmt.Errorf("ceremony: %s callout xkey does not match %s", fileAuthJWT, fileCallout)
	}
	if !auth.SigningKeys.Contains(s.AuthSigningPub) {
		return nil, fmt.Errorf("ceremony: %s does not endorse %s", fileAuthJWT, fileAuthSigning)
	}
	realm, err := jwt.DecodeAccountClaims(s.RealmJWT)
	if err != nil || realm.Subject != s.RealmPub {
		return nil, fmt.Errorf("ceremony: %s does not match %s", fileRealmJWT, fileRealmSeed)
	}
	if scope, ok := realm.SigningKeys.GetScope(s.RealmSigningPub); !ok || scope == nil {
		return nil, fmt.Errorf("ceremony: %s lacks the scoped signing key %s", fileRealmJWT, fileRealmSigning)
	}
	// The workload minting key must be endorsed PLAIN (nil scope): a
	// scoped key would reject the minter's carried permissions (R1).
	if scope, ok := realm.SigningKeys.GetScope(s.WorkloadSigningPub); !ok || scope != nil {
		return nil, fmt.Errorf("ceremony: %s does not endorse %s as a plain signing key", fileRealmJWT, fileWorkloadSigning)
	}
	host, _, err := net.SplitHostPort(s.Listen)
	if err != nil {
		return nil, fmt.Errorf("ceremony: damaged %s: listen %q: %w", fileConfig, s.Listen, err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, fmt.Errorf("ceremony: %s listen %q is not loopback", fileConfig, s.Listen)
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
		// Sessions need a sign-in issuer: the bundled fold, or an
		// external AS via planes.mcp.auth_issuer.
		if s.MCPAuthIssuer == "" && !s.SignInEnabled {
			return nil, fmt.Errorf("ceremony: %s enables the shell plane with no sign-in issuer — enable planes.signin or set planes.mcp.auth_issuer", fileConfig)
		}
	}
	return s, nil
}

// ArtifactCount is the number of persisted founding artifacts a complete
// state directory carries (config + keys + users + sentinel) — the number
// init's verify report cites.
func ArtifactCount() int { return 19 + 1 + 1 } // files() + config + sentinel

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
