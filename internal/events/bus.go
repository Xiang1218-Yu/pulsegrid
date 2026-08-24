package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"pulsegrid/internal/domain"
)

var ErrClosed = errors.New("event bus is closed")

type Config struct {
	Buffer       int
	DropWhenBusy bool
}

type Bus struct {
	mu     sync.RWMutex
	config Config
	closed bool
	next   atomic.Uint64
	subs   map[uint64]*subscription
	topics map[string]map[uint64]bool
}

// subscription holds a single subscriber's state. Its own mutex serializes
// delivery (send) against teardown (close): a Publish that has already
// snapshotted this subscription finishes its send, or notices the channel has
// been closed, without ever racing a concurrent close. That is what keeps a
// subscriber disconnecting mid-stream from panicking the publisher — and with
// it the whole process and every other in-flight monitor request.
type subscription struct {
	id      uint64
	bus     *Bus
	topics  map[string]bool
	mu      sync.Mutex
	closed  bool
	channel chan domain.Event
	once    sync.Once
}

type Subscription struct {
	value *subscription
}

func New(config Config) *Bus {
	if config.Buffer < 1 {
		config.Buffer = 16
	}
	return &Bus{config: config, subs: map[uint64]*subscription{}, topics: map[string]map[uint64]bool{}}
}

func (b *Bus) Subscribe(topics ...string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.next.Add(1)
	value := &subscription{id: id, bus: b, topics: map[string]bool{}, channel: make(chan domain.Event, b.config.Buffer)}
	if len(topics) == 0 {
		topics = []string{"*"}
	}
	for _, topic := range topics {
		if topic == "" {
			topic = "*"
		}
		value.topics[topic] = true
		if b.topics[topic] == nil {
			b.topics[topic] = map[uint64]bool{}
		}
		b.topics[topic][id] = true
	}
	b.subs[id] = value
	return &Subscription{value: value}
}

func (s *Subscription) Events() <-chan domain.Event { return s.value.channel }

// Close tears the subscription down: it removes the subscription from the bus
// registry and closes its channel. Removal is essential — without it the bus
// keeps a stale reference whose channel is already closed, and the next
// Publish would send on a closed channel and panic, taking the process down
// with it. Close is safe to call more than once.
func (s *Subscription) Close() {
	s.value.once.Do(func() {
		s.value.bus.remove(s.value.id)
	})
}

func (b *Bus) Publish(ctx context.Context, event domain.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrClosed
	}
	recipients := b.recipientsLocked(event.Type)
	drop := b.config.DropWhenBusy
	b.mu.RUnlock()
	for _, recipient := range recipients {
		// Deliver under the subscription's own lock so that a concurrent
		// Close (which closes the channel under the same lock) can never
		// overlap a send. If the subscriber has torn down, we simply skip it
		// — no send on a closed channel, no panic, no process crash.
		recipient.mu.Lock()
		if recipient.closed {
			recipient.mu.Unlock()
			continue
		}
		if drop {
			select {
			case recipient.channel <- event:
			default:
			}
			recipient.mu.Unlock()
			continue
		}
		// Release the per-subscription lock while blocking on a slow
		// subscriber only if the context is already done; otherwise keep it
		// to stay race-free against Close.
		select {
		case <-ctx.Done():
			recipient.mu.Unlock()
			return ctx.Err()
		case recipient.channel <- event:
		}
		recipient.mu.Unlock()
	}
	return nil
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, sub := range b.subs {
		sub.mu.Lock()
		sub.closed = true
		close(sub.channel)
		sub.mu.Unlock()
	}
	b.subs = map[uint64]*subscription{}
	b.topics = map[string]map[uint64]bool{}
}

// remove unregisters a subscription from the bus and closes its channel.
// Called from Subscription.Close via the per-subscription Once, so it runs at
// most once per subscription.
func (b *Bus) remove(id uint64) {
	b.mu.Lock()
	sub, ok := b.subs[id]
	if !ok {
		b.mu.Unlock()
		return
	}
	delete(b.subs, id)
	for topic := range sub.topics {
		delete(b.topics[topic], id)
	}
	b.mu.Unlock()
	// Close the channel outside the bus lock so a Publish that already holds
	// the subscription lock (and is mid-send) is not blocked from finishing;
	// the lock here serializes the close against any in-flight send on this
	// same subscription.
	sub.mu.Lock()
	sub.closed = true
	close(sub.channel)
	sub.mu.Unlock()
}

func (b *Bus) recipientsLocked(topic string) []*subscription {
	ids := map[uint64]bool{}
	for id := range b.topics[topic] {
		ids[id] = true
	}
	for id := range b.topics["*"] {
		ids[id] = true
	}
	result := make([]*subscription, 0, len(ids))
	for id := range ids {
		if sub, ok := b.subs[id]; ok {
			result = append(result, sub)
		}
	}
	return result
}

func NewEvent(organizationID, kind, subject, contactID string, data map[string]any) domain.Event {
	return domain.Event{
		ID:             "event-" + time.Now().UTC().Format("20060102T150405.000000000"),
		OrganizationID: organizationID, Type: kind, Subject: subject, ContactID: contactID,
		Data: cloneData(data), OccurredAt: time.Now().UTC(),
	}
}

func cloneData(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}
