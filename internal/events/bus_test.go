package events

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pulsegrid/internal/domain"
)

// TestSubscribeCloseThenPublishNoPanic reproduces the resource-lifetime bug:
// a client subscribes, the connection ends (Subscription.Close is called),
// and then events keep arriving. Before the fix the bus still held the stale
// subscription and Publish sent on its closed channel, panicking — which
// killed the whole process and every other in-flight monitor request.
func TestSubscribeCloseThenPublishNoPanic(t *testing.T) {
	bus := New(Config{Buffer: 4, DropWhenBusy: true})

	// Subscriber that disconnects immediately.
	gone := bus.Subscribe("contact.created")
	gone.Close()

	// A healthy subscriber that is still listening.
	live := bus.Subscribe("contact.created")
	defer live.Close()

	// Drain the live subscriber so Publish never blocks on a full buffer.
	var drained sync.WaitGroup
	drained.Add(1)
	var received atomic.Int64
	go func() {
		defer drained.Done()
		for range live.Events() {
			received.Add(1)
		}
	}()

	// Publish after the disconnect. This used to panic.
	for i := 0; i < 100; i++ {
		if err := bus.Publish(context.Background(), domain.Event{Type: "contact.created", ID: "e"}); err != nil {
			t.Fatalf("publish failed: %v", err)
		}
	}

	// The live subscriber must have received at least one event.
	if received.Load() == 0 {
		t.Fatalf("live subscriber did not receive any events")
	}
}

// TestCloseUnregistersFromBus ensures Close removes the subscription from the
// bus registry so the channel is not kept alive and cannot be sent on later.
func TestCloseUnregistersFromBus(t *testing.T) {
	bus := New(Config{Buffer: 4, DropWhenBusy: false})
	sub := bus.Subscribe("contact.created")
	sub.Close()

	bus.mu.RLock()
	_, inSubs := bus.subs[sub.value.id]
	_, inTopics := bus.topics["contact.created"][sub.value.id]
	bus.mu.RUnlock()

	if inSubs {
		t.Fatalf("subscription still registered in subs after Close")
	}
	if inTopics {
		t.Fatalf("subscription still registered in topics after Close")
	}
}

// TestCloseIsIdempotent ensures Close can be called repeatedly without a
// "close of closed channel" panic.
func TestCloseIsIdempotent(t *testing.T) {
	bus := New(Config{Buffer: 4})
	sub := bus.Subscribe("contact.created")
	for i := 0; i < 5; i++ {
		sub.Close()
	}
}

// TestConcurrentCloseAndPublish stresses the race between Close and Publish to
// make sure the publisher never panics under concurrent teardown.
func TestConcurrentCloseAndPublish(t *testing.T) {
	bus := New(Config{Buffer: 1, DropWhenBusy: true})

	const publishers = 8
	const subscribers = 40
	var wg sync.WaitGroup

	wg.Add(publishers)
	for p := 0; p < publishers; p++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_ = bus.Publish(context.Background(), domain.Event{Type: "contact.created"})
			}
		}()
	}

	wg.Add(subscribers)
	for s := 0; s < subscribers; s++ {
		go func() {
			defer wg.Done()
			sub := bus.Subscribe("contact.created")
			// Drain a little, then disconnect mid-stream.
			time.Sleep(time.Microsecond)
			sub.Close()
		}()
	}

	wg.Wait()
}

// TestPublishAfterDisconnectNoCrash mirrors the production configuration
// (DropWhenBusy: true, as wired in cmd/pulsegridd) and the lifecycle reported
// in the bug: a monitor client subscribes, disconnects, and events keep
// arriving. Before the fix the worker/request goroutine that called Publish
// panicked on the closed channel and took the whole process down. This test
// publishes from a goroutine with no recover (just like a jobs worker) and
// asserts the goroutine returns normally rather than killing the process.
func TestPublishAfterDisconnectNoCrash(t *testing.T) {
	bus := New(Config{Buffer: 8, DropWhenBusy: true})

	// A subscriber that disconnects right away.
	gone := bus.Subscribe("contact.created")
	gone.Close()

	// Publish from a bare goroutine with no recover — the worker shape.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			_ = bus.Publish(context.Background(), domain.Event{Type: "contact.created"})
		}
	}()

	select {
	case <-done:
		// Publisher completed without panicking.
	case <-time.After(5 * time.Second):
		t.Fatal("publisher goroutine hung; close/publish deadlock")
	}
}
