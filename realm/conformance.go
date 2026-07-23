package realm

import (
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

// streamNonconformities compares an existing op-log stream's configuration to the
// mandated shape and returns one message per drift. An empty result means conformant.
//
// History-risking drift — age-based expiry present, or rollup disabled — is reported
// like any other drift and never silently corrected: changing those settings on a
// live stream can destroy history, so the decision is left to the operator. (The one
// exception is the recognised legacy subject shape, which provisioning converges
// deliberately — see provisionStream.)
func streamNonconformities(cfg jetstream.StreamConfig) []string {
	var issues []string

	if len(cfg.Subjects) != 1 || cfg.Subjects[0] != StreamSubject {
		issues = append(issues, fmt.Sprintf("subjects are %v, want [%s]", cfg.Subjects, StreamSubject))
	}
	if cfg.Retention != jetstream.LimitsPolicy {
		issues = append(issues, "retention is not Limits")
	}
	if cfg.MaxAge != 0 {
		issues = append(issues, fmt.Sprintf("MaxAge is set to %s (age-based expiry present)", cfg.MaxAge))
	}
	if !cfg.AllowRollup {
		issues = append(issues, "AllowRollup (allow_rollup_hdrs) is disabled")
	}
	if cfg.Duplicates < MinDuplicateWindow {
		issues = append(issues, fmt.Sprintf("duplicate_window is %s, want at least %s", cfg.Duplicates, MinDuplicateWindow))
	}
	if cfg.Storage != jetstream.FileStorage {
		issues = append(issues, "storage is not File")
	}

	return issues
}

// notifyNonconformities compares an existing inbox stream's configuration to the
// mandated shape. Same stance as the op-log: report, never mutate.
func notifyNonconformities(cfg jetstream.StreamConfig) []string {
	var issues []string

	if len(cfg.Subjects) != 1 || cfg.Subjects[0] != NotifyStreamSubject {
		issues = append(issues, fmt.Sprintf("subjects are %v, want [%s]", cfg.Subjects, NotifyStreamSubject))
	}
	if cfg.Retention != jetstream.LimitsPolicy {
		issues = append(issues, "retention is not Limits")
	}
	if cfg.MaxAge != 0 {
		issues = append(issues, fmt.Sprintf("MaxAge is set to %s (age-based expiry present)", cfg.MaxAge))
	}
	if cfg.MaxMsgsPerSubject != InboxWindow {
		issues = append(issues, fmt.Sprintf("MaxMsgsPerSubject is %d, want %d (the inbox window)", cfg.MaxMsgsPerSubject, InboxWindow))
	}
	if cfg.Discard != jetstream.DiscardOld {
		issues = append(issues, "discard policy is not Old (a full inbox must shed its oldest notification)")
	}
	if cfg.Duplicates < MinDuplicateWindow {
		issues = append(issues, fmt.Sprintf("duplicate_window is %s, want at least %s", cfg.Duplicates, MinDuplicateWindow))
	}
	if cfg.Storage != jetstream.FileStorage {
		issues = append(issues, "storage is not File")
	}

	return issues
}
