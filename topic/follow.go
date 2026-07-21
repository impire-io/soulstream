package topic

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/record"
)

// Follow materialises the topic, then keeps applying live operations, calling onOp with
// the updated view after each applied op. It uses a single ordered consumer that
// delivers history then live messages in one continuous stream, so there is no
// replay/live seam. It blocks until ctx is cancelled, then returns nil.
func (h *Handle) Follow(ctx context.Context, onOp func(*MaterializedTopic)) error {
	stream, err := h.client.JetStream().Stream(ctx, realm.StreamName)
	if err != nil {
		return fmt.Errorf("topic: look up stream: %w", err)
	}

	cons, err := stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{OpsSubject(h.path)},
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
		stop() // unblocks Next()
	}()

	var recs []SeqRecord
	for {
		msg, err := it.Next()
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) {
				if ctx.Err() != nil {
					return nil // cancelled — normal termination
				}
				return err
			}
			return fmt.Errorf("topic: read op: %w", err)
		}

		md, err := msg.Metadata()
		if err != nil {
			return fmt.Errorf("topic: message metadata: %w", err)
		}
		rec, perr := record.Parse(msg.Headers(), msg.Data())
		if perr != nil {
			continue // skip an unparseable op
		}

		recs = append(recs, SeqRecord{Record: rec, StreamSeq: md.Sequence.Stream})
		mt := apply(h.path, recs)
		annotateView(mt, annotate(recs, h.client.Realm(), h.path, h.keyring), recs[0].Record.ID)
		h.adopt(mt)
		if onOp != nil {
			onOp(mt)
		}
	}
}
