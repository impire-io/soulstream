package topic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/record"
)

// NotifySubjectPrefix is the prefix of a persona's notify (inbox) subject.
const NotifySubjectPrefix = "SOULSTREAM.PERSONA.NOTIFY."

// NotifySubject returns a persona's notify subject.
func NotifySubject(persona string) string { return NotifySubjectPrefix + persona }

// Notification is a received mention.notify.
type Notification struct {
	Topic  string
	OpID   string
	Author string
}

// publishNotify publishes a mention.notify record to a persona's inbox.
func publishNotify(ctx context.Context, c *realm.Client, persona string, payload NotifyPayload) error {
	if _, err := publishOp(ctx, c, NotifySubject(persona), TypeMentionNotify, payload, nil); err != nil {
		return fmt.Errorf("topic: notify %s: %w", persona, err)
	}
	return nil
}

// FetchInbox returns the persona's mention notifications, newest-first, capped at limit
// (a limit of 0 or less means the default of 50). It returns an empty slice (no error)
// when the inbox is empty. Unlike FollowInbox, it is a bounded one-shot read — the shape
// a request/response caller (such as the MCP adapter) needs.
func FetchInbox(ctx context.Context, c *realm.Client, persona string, limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}

	stream, err := c.JetStream().Stream(ctx, realm.StreamName)
	if err != nil {
		return nil, fmt.Errorf("topic: look up stream: %w", err)
	}
	subject := NotifySubject(persona)

	// Empty guard (an ordered consumer's Next() would otherwise block on an empty inbox).
	if _, err := stream.GetLastMsgForSubject(ctx, subject); err != nil {
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("topic: probe inbox: %w", err)
	}

	cons, err := stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{subject},
		DeliverPolicy:  jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("topic: create consumer: %w", err)
	}
	it, err := cons.Messages()
	if err != nil {
		return nil, fmt.Errorf("topic: consume: %w", err)
	}
	defer it.Stop()

	var all []Notification
	for {
		msg, err := it.Next()
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) {
				break
			}
			return nil, fmt.Errorf("topic: read inbox: %w", err)
		}
		md, err := msg.Metadata()
		if err != nil {
			return nil, fmt.Errorf("topic: message metadata: %w", err)
		}
		if rec, perr := record.Parse(msg.Headers(), msg.Data()); perr == nil && rec.Type == TypeMentionNotify {
			var np NotifyPayload
			if json.Unmarshal(rec.Payload, &np) == nil {
				all = append(all, Notification(np))
			}
		}
		if md.NumPending == 0 {
			break
		}
	}

	// Newest-first, then cap at the limit.
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// FollowInbox subscribes to persona's notify subject and calls onNotify for each
// mention.notify (history then live). It blocks until ctx is cancelled, then returns nil.
func FollowInbox(ctx context.Context, c *realm.Client, persona string, onNotify func(Notification)) error {
	stream, err := c.JetStream().Stream(ctx, realm.StreamName)
	if err != nil {
		return fmt.Errorf("topic: look up stream: %w", err)
	}

	cons, err := stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{NotifySubject(persona)},
		DeliverPolicy:  jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return fmt.Errorf("topic: create consumer: %w", err)
	}
	it, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("topic: consume: %w", err)
	}

	var stopOnce sync.Once
	stop := func() { stopOnce.Do(it.Stop) }
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()

	for {
		msg, err := it.Next()
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			return fmt.Errorf("topic: read notify: %w", err)
		}

		rec, perr := record.Parse(msg.Headers(), msg.Data())
		if perr != nil || rec.Type != TypeMentionNotify {
			continue
		}
		var np NotifyPayload
		if json.Unmarshal(rec.Payload, &np) != nil {
			continue
		}
		if onNotify != nil {
			onNotify(Notification(np))
		}
	}
}
