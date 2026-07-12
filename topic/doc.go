// Package topic is the Soulstream op-log engine: it turns operation records into
// topics — the shared workbenches personas collaborate on.
//
// A topic is an append-only op-log on its own subject. Starting a topic publishes an
// announcement (so others can find it) and an initial baseline (the zero-point its
// operations build on). Personas contribute operations through a [Handle]; anyone can
// rebuild the current state by replaying the log ([Handle.Materialise]) or follow it
// live ([Handle.Follow]); anyone can list the realm's topics with [Board].
//
// The projection logic — folding a sequence of records into a [MaterializedTopic], and
// the board — is a pure function of the log, kept free of NATS so it unit-tests with no
// server. Ordering is by JetStream stream sequence this cycle; the DAG (parents) is
// recorded but not consulted (eg-walker merge is deferred).
package topic
