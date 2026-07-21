# Quickstart: Topics

*What you can do once this cycle is built: start a topic, talk in it, follow it live, close it,
nest sub-topics, and list the board.*

## Prerequisites

- `001-foundation` merged; a provisioned realm (see its quickstart) reachable via a named context.

## 1. Start a topic

```go
import (
    "context"

    "github.com/impire-io/soulstream/realm"
    "github.com/impire-io/soulstream/topic"
)

ctx := context.Background()
c, _ := realm.Connect(ctx, realm.Config{ContextName: "soulstream", Realm: "acme", Persona: "daan"})
defer c.Close()

h, err := topic.StartTopic(ctx, c, topic.StartTopicInput{
    Name:          "Q2 VAT filing",
    SubjectMatter: "Preparing and checking the Q2 2026 VAT return.",
    Tags:          []string{"finance", "recurring"},
    Expected:      []string{"daan", "bookkeeper-agent"},
})
// h.Path() == "q2-vat-filing-<suffix>" ; announce on INFO, initial baseline on OPS.
```

## 2. Talk, comment, close

```go
turnID, _ := h.PostTurn(ctx, "Numbers are in — @bookkeeper-agent can you sanity-check box 5?")
_, _ = h.AddComment(ctx, "Box 5 looks off by the reverse-charge total.", turnID)
_, _ = h.Transition(ctx, topic.Closed) // records a life.transition; lifecycle becomes "closed"
```

`PostTurn`/`AddComment`/`Transition` stamp the author (`daan`), fill parents from the frontier the
handle has seen, generate the op-id, and publish. Posting a content op to a closed topic is
**warned, not blocked** — closing is a convention.

## 3. Materialise (see where the topic stands)

```go
view, _ := h.Materialise(ctx)
// view.Lifecycle == topic.Closed
// view.Contributions ordered by stream sequence; the comment records Anchor == turnID
// view.Frontier == the current leaf op-ids
for _, cont := range view.Contributions {
    fmt.Printf("%s  %-11s %s\n", cont.Author, cont.Type, cont.Body)
}
```

Materialise is a pure replay: two people replaying the same log see the identical view.

## 4. Follow live

```go
// From another process/persona:
other := topic.Open(c2, h.Path())
go other.Follow(ctx, func(v *topic.MaterializedTopic) {
    fmt.Println("topic changed; now", len(v.Contributions), "contributions")
})
// Meanwhile someone posts — the follower's callback fires with the updated view, no re-replay.
```

Follow uses one ordered consumer: it replays history, then keeps delivering new ops — no seam.

## 5. Sub-topics

```go
sub, _ := topic.StartTopic(ctx, c, topic.StartTopicInput{
    Name:   "Pricing angle",
    Parent: h.Path(), // nests under the VAT topic
})
// sub.Path() == "q2-vat-filing-<suffix>.pricing-angle-<suffix>"
// its ops live at SOULSTREAM.TOPICS.OPS.q2-vat-filing-<suffix>.pricing-angle-<suffix>
```

## 6. Discover the board

```go
board, _ := topic.Board(ctx, c)
for _, e := range board {
    fmt.Printf("%-40s %-9s parent=%q\n", e.Path, e.Lifecycle, e.Parent)
}
// One entry per topic (latest announcement), sub-topics show their parent; empty realm → empty board.
```

## Verify (definition of done)

```sh
make check   # fmt + tidy + build + test + lint, all green, none skipped
```

Pure materialisation/board rules are tested by feeding synthetic records (no server); publish,
replay, follow, and discovery are tested against an in-process JetStream server.
