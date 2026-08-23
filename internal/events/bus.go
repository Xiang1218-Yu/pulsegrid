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

type subscription struct {
	id      uint64
	bus     *Bus
	topics  map[string]bool
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

func (s *Subscription) Close() { s.value.once.Do(func() { s.value.bus.remove(s.value.id) }) }

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
		if drop {
			select {
			case recipient.channel <- event:
			default:
			}
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case recipient.channel <- event:
		}
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
		close(sub.channel)
	}
	b.subs = map[uint64]*subscription{}
	b.topics = map[string]map[uint64]bool{}
}

func (b *Bus) remove(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub, ok := b.subs[id]
	if !ok {
		return
	}
	delete(b.subs, id)
	for topic := range sub.topics {
		delete(b.topics[topic], id)
	}
	close(sub.channel)
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
	eventData := cloneData(data)
	delete(eventData, "contact_id")
	return domain.Event{
		ID:             "event-" + time.Now().UTC().Format("20060102T150405.000000000"),
		OrganizationID: organizationID, Type: kind, Subject: subject, ContactID: contactID,
		Data: eventData, OccurredAt: time.Now().UTC(),
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
