package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"pulsegrid/internal/domain"
)

// safePublisher publishes events to the bus until stopped, recovering any panic
// so the test can report a failure instead of taking the whole process down.
func safePublisher(t *testing.T, bus *Bus, topic string, stop <-chan struct{}) *sync.WaitGroup {
	t.Helper()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("publisher panicked: %v", r)
			}
		}()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			i++
			_ = bus.Publish(context.Background(), domain.Event{ID: "e", Type: topic, Data: map[string]any{"n": i}})
		}
	}()
	return &wg
}

// TestSubscribeCloseDuringBlockingSend reproduces the original crash: a publisher
// blocked on a full, non-drop subscription channel while the subscriber closes
// the subscription. Before the fix, remove() closed the channel without holding
// sendMu, racing with send()'s `channel <- event` and panicking with
// "send on closed channel".
func TestSubscribeCloseDuringBlockingSend(t *testing.T) {
	bus := New(Config{Buffer: 1, DropWhenBusy: false})
	const topic = "delivery.opened"
	stop := make(chan struct{})
	wg := safePublisher(t, bus, topic, stop)

	sub := bus.Subscribe(topic)

	// Start a goroutine that holds the event in the buffer without draining,
	// so subsequent publishes block inside send on the full channel.
	received := make(chan domain.Event, 4)
	go func() {
		for ev := range sub.Events() {
			received <- ev
		}
		close(received)
	}()

	// Give the publisher time to fill the buffer and block on the next send.
	time.Sleep(50 * time.Millisecond)

	// Close while a publish is mid-send. This must not panic.
	sub.Close()

	close(stop)
	wg.Wait()
}

// TestSubscribeCloseDuringDropSend reproduces the drop-mode race: remove()
// closed channel without sendMu while a drop-mode send was selecting on
// `channel <- event`.
func TestSubscribeCloseDuringDropSend(t *testing.T) {
	bus := New(Config{Buffer: 1, DropWhenBusy: true})
	const topic = "delivery.clicked"
	stop := make(chan struct{})
	wg := safePublisher(t, bus, topic, stop)

	sub := bus.Subscribe(topic)

	// Don't drain, so the buffer stays full and every publish hits the drop
	// select that races with close.
	time.Sleep(50 * time.Millisecond)
	sub.Close()

	close(stop)
	wg.Wait()
}

// TestSubscribeReceivesEvents confirms the normal subscription path still
// delivers events to a draining subscriber.
func TestSubscribeReceivesEvents(t *testing.T) {
	bus := New(Config{Buffer: 4, DropWhenBusy: false})
	const topic = "contact.subscribed"
	sub := bus.Subscribe(topic)

	events := make([]domain.Event, 0, 3)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range sub.Events() {
			events = append(events, ev)
			if len(events) == 3 {
				sub.Close()
			}
		}
	}()

	for i := 1; i <= 3; i++ {
		if err := bus.Publish(context.Background(), domain.Event{ID: "e", Type: topic, Data: map[string]any{"n": i}}); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for events")
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

// TestBusCloseDuringSend ensures shutting the whole bus down while publishers
// are active does not panic.
func TestBusCloseDuringSend(t *testing.T) {
	bus := New(Config{Buffer: 1, DropWhenBusy: false})
	const topic = "organization.created"
	stop := make(chan struct{})
	wg := safePublisher(t, bus, topic, stop)
	sub := bus.Subscribe(topic)

	// Drain slowly so the channel backs up and publishes block.
	go func() {
		for range sub.Events() {
			time.Sleep(time.Millisecond)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	bus.Close()

	close(stop)
	wg.Wait()
}
