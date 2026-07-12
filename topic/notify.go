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
