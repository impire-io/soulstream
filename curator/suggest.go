package curator

import (
	"fmt"
	"strings"
	"time"
)

// The suggestion body convention: a stable, visible marker so personas — and other
// curators — can tell suggestion from conversation. Recognition is by prefix and
// author-independent: any persona writing the marker is making the same suggestion.
const (
	// SuggestionDuplicatePrefix opens a duplicate flag; the older topic's path
	// follows it.
	SuggestionDuplicatePrefix = "[curator] this looks similar to "
	// SuggestionDormantPrefix opens a dormancy proposal; the idle span follows it.
	SuggestionDormantPrefix = "[curator] no activity for "
)

// DuplicateSuggestion builds the one comment a curator leaves in the newer of two
// look-alike topics.
func DuplicateSuggestion(olderPath string) string {
	return SuggestionDuplicatePrefix + olderPath + " — consider continuing there"
}

// DormantSuggestion builds the one comment a curator leaves in a topic that has
// been quiet past the idle window.
func DormantSuggestion(idle time.Duration) string {
	return fmt.Sprintf("%s%s — close it if it's done", SuggestionDormantPrefix, humanSpan(idle))
}

// IsSuggestion reports whether body is a curator suggestion of either kind.
func IsSuggestion(body string) bool {
	return IsDuplicateSuggestion(body) || IsDormantSuggestion(body)
}

// IsDuplicateSuggestion reports whether body is a duplicate flag.
func IsDuplicateSuggestion(body string) bool {
	return strings.HasPrefix(body, SuggestionDuplicatePrefix)
}

// IsDormantSuggestion reports whether body is a dormancy proposal.
func IsDormantSuggestion(body string) bool {
	return strings.HasPrefix(body, SuggestionDormantPrefix)
}

// humanSpan renders a duration the way a person would say it in a comment.
func humanSpan(d time.Duration) string {
	switch {
	case d >= 48*time.Hour:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	case d >= 2*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	case d >= 2*time.Minute:
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	default:
		return d.String()
	}
}
