// Package curator implements the curation extension: keeping an active realm
// liveable as a persona habit, never a protocol role.
//
// A curator is an ordinary persona — ordinary credentials, ordinary (signed)
// operations, zero protocol standing — that maintains a warm, content-aware
// projection of the realm's topics and uses it for three habits: answering
// discovery better than a cold responder can, flagging likely duplicate topics,
// and proposing closure for long-dormant ones. It suggests, never enforces:
// comments are its entire vocabulary of action, and the log itself is its memory
// (a suggestion already in a topic is never repeated — across restarts and across
// curators alike).
//
// The package deliberately consumes only the library's public surfaces. Nothing in
// the realm knows curators exist: run none and everything works, run several and
// they cooperate by reading each other's comments, stop one with a plain context
// cancel and nothing needs deregistering.
package curator
