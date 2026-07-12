# Quickstart: Participation

*Once built: mention people, watch an inbox, attach and retrieve files.*

## Mentions

```go
// Posting with an @mention records it and pings that persona's inbox.
opID, _ := h.PostTurn(ctx, "numbers are in, @bookkeeper-agent please check box 5")

// The mentioned persona (its own client) watches its inbox:
go topic.FollowInbox(ctx, agentClient, "bookkeeper-agent", func(n topic.Notification) {
    fmt.Printf("mentioned in %s (op %s) by %s\n", n.Topic, n.OpID, n.Author)
    // → read the op and reply
})
```

`ParseMentions("@Daan @@ @ x @daan @daan")` → `["daan"]` (invalid/duplicate dropped).

## Attachments

```go
data := []byte("id,amount\n1,100\n")
opID, _ := h.Attach(ctx, "q2-lines.csv", "text/csv", data, "" /*anchor*/)

// It shows up when the topic is materialised:
view, _ := h.Materialise(ctx)
a := view.Attachments[0]           // Name, Object, Digest, Size, ContentType, Anchor

// Anyone can fetch it back and verify:
got, _ := topic.GetAttachment(ctx, c, a.Object)
ok := topic.VerifyDigest(got, a.Digest) // true — the reference is verifiable
```

## Verify (definition of done)

```sh
make check   # fmt + tidy + build + test + lint, all green, none skipped
```

Mention parsing and digest verification are tested with no server; notify delivery and attachment
put/materialise/get are tested against an in-process JetStream server.
