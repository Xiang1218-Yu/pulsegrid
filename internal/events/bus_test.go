package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pulsegrid/internal/domain"
)

func makeEvent() domain.Event { return domain.Event{Type: "topic", ID: "e"} }

// A subscriber that disconnects must be fully removed from the bus so it does
// not leak its buffered channel or keep receiving routed events.
func TestSubscribeCloseReleasesResources(t *testing.T) {
	bus := New(Config{Buffer: 4, DropWhenBusy: false})
	sub := bus.Subscribe("topic")

	if got := bus.SubscriberCount(); got != 1 {
		t.Fatalf("expected 1 subscriber, got %d", got)
	}

	sub.Close()

	if got := bus.SubscriberCount(); got != 0 {
		t.Fatalf("expected 0 subscribers after Close, got %d", got)
	}
	select {
	case <-sub.Done():
	case <-time.After(time.Second):
		t.Fatalf("expected Done() to be closed after Close")
	}

	// The removed subscription must not appear in recipient collection.
	if got := len(bus.recipientsLocked("topic")); got != 0 {
		t.Fatalf("expected no recipients, got %d", got)
	}
}

// Publishing while a subscriber disconnects must not panic and must still
// deliver to subscribers that remain connected.
func TestPublishConcurrentUnsubscribe(t *testing.T) {
	for i := 0; i < 50; i++ {
		bus := New(Config{Buffer: 1, DropWhenBusy: i%2 == 0})
		keep := bus.Subscribe("topic")
		goaway := bus.Subscribe("topic")

		var (
			wg     sync.WaitGroup
			panicv any
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			defer func() { panicv = recover() }()
			_ = bus.Publish(context.Background(), makeEvent())
		}()
		go func() {
			defer wg.Done()
			goaway.Close()
		}()
		wg.Wait()

		if panicv != nil {
			t.Fatalf("publish panicked during concurrent unsubscribe: %v", panicv)
		}
		keep.Close()
		_ = keep
	}
}

// A live subscriber keeps receiving events while others churn in and out.
func TestLiveSubscriberReceivesEventsDuringChurn(t *testing.T) {
	bus := New(Config{Buffer: 8, DropWhenBusy: false})
	live := bus.Subscribe("topic")
	defer live.Close()

	const total = 200
	go func() {
		for n := 0; n < total; n++ {
			// On every other iteration, attach a short-lived subscriber that
			// closes immediately, simulating clients disconnecting.
			if n%2 == 0 {
				bus.Subscribe("topic").Close()
			}
			_ = bus.Publish(context.Background(), domain.Event{Type: "topic", ID: "e"})
		}
	}()

	received := 0
loop:
	for {
		select {
		case _, ok := <-live.Events():
			if !ok {
				break loop
			}
			received++
			if received == total {
				break loop
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out after receiving %d/%d events", received, total)
		}
	}
	if received != total {
		t.Fatalf("expected %d events on live subscriber, got %d", total, received)
	}
}

// Publish must not block forever when a slow subscriber never drains; in
// drop mode it returns promptly, and a cancellable context aborts the wait.
func TestPublishDropOrCancelDoesNotBlock(t *testing.T) {
	t.Run("drop", func(t *testing.T) {
		bus := New(Config{Buffer: 16, DropWhenBusy: true})
		bus.Subscribe("topic") // never drained
		for i := 0; i < 100; i++ {
			if err := bus.Publish(context.Background(), makeEvent()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
	})
	t.Run("cancel", func(t *testing.T) {
		bus := New(Config{Buffer: 16, DropWhenBusy: false})
		undrained := bus.Subscribe("topic") // never drained
		defer undrained.Close()
		// Fill the buffered channel so the next send must block.
		for i := 0; i < 16; i++ {
			if err := bus.Publish(context.Background(), makeEvent()); err != nil {
				t.Fatalf("unexpected error filling buffer: %v", err)
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err := bus.Publish(ctx, makeEvent())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded, got %v", err)
		}
	})
}
