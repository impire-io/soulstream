package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/impire-io/soulstream/ceremony"
	"github.com/impire-io/soulstream/internal/synadia"
)

// synadiaAccountHalf drives the Synadia Cloud account half (design 0003
// §5) and persists what came back — immediately, because the
// programmatic seeds are returned exactly once. The PAT arrives by
// environment and is never persisted; the state dir's custody audit
// holds (spec 010 SC-002).
func synadiaAccountHalf(st *ceremony.State, dir string, errw io.Writer) error {
	token := os.Getenv("SOULSTREAM_SYNADIA_TOKEN")
	if token == "" {
		return fmt.Errorf("SOULSTREAM_SYNADIA_TOKEN is required for the synadia-cloud flavour — mint a personal access token in Synadia Cloud and export it; it drives the account half and is never persisted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	scopePub, scopeSub := ceremony.PersonaScopeAllows()
	res, err := synadia.Setup(ctx, synadia.Config{
		BaseURL: os.Getenv("SOULSTREAM_SYNADIA_URL"), // test seam; default cloud.synadia.com
		Token:   token,
		System:  st.SynadiaSystem,
		Realm:   st.Realm,
		Existing: synadia.Existing{
			RealmScopedSeed: st.RealmSigningSeed,
			WorkloadSeed:    st.WorkloadSigningSeed,
			AuthSigningSeed: st.AuthSigningSeed,
		},
		// Each once-returned seed lands on disk before the driver takes
		// another step — a mid-run tunnel cycle can never orphan a group
		// (measured 2026-08-16).
		OnSeed: func(group string, seed []byte) error {
			switch group {
			case synadia.GroupScoped:
				st.RealmSigningSeed = seed
			case synadia.GroupWorkload:
				st.WorkloadSigningSeed = seed
			case synadia.GroupAuth:
				st.AuthSigningSeed = seed
			default:
				return fmt.Errorf("unknown signing-key group %q", group)
			}
			return st.Save(dir)
		},
		Log: errw,
	}, scopePub, scopeSub)
	if err != nil {
		return err
	}
	st.AuthPub = res.AuthAccountPub
	st.RealmPub = res.RealmAccountPub
	st.RealmSigningSeed = res.RealmScopedSeed
	st.WorkloadSigningSeed = res.WorkloadSeed
	st.AuthSigningSeed = res.AuthSigningSeed
	st.IssuerCreds = res.IssuerCreds
	if res.XkeyPublic != "" {
		fmt.Fprintf(errw, "soulstream: the platform custodies the callout xkey (%s) — requests are sealed to a key this deployment does not hold, so the callout runs unsealed on our side (design 0003 §5, said out loud)\n", res.XkeyPublic)
	}
	return st.Save(dir)
}
