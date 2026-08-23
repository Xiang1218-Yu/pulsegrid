package events

import (
	"context"
	"testing"
	"time"
)

// Bus.Publish, Subscription.Close, events handler
func TestBug026EventSubscriptionClose(t *testing.T) {
	bus := New(Config{Buffer: 1, DropWhenBusy: false})
	subscription := bus.Subscribe("delivery.sent")
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("closing a subscription must not panic an in-flight publisher: %v", recovered)
		}
	}()

	event := NewEvent("org-1", "delivery.sent", "delivery-1", "contact-1", nil)
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("fill subscription buffer: %v", err)
	}

	publisherDone := make(chan struct{})
	go func() {
		defer close(publisherDone)
		_ = bus.Publish(context.Background(), event)
	}()
	time.Sleep(2 * time.Millisecond)
	subscription.Close()

	select {
	case <-publisherDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("publisher remained blocked after subscription close")
	}
}

func TestBug026EventPublishRegression(t *testing.T) {
	bus := New(Config{Buffer: 2, DropWhenBusy: true})
	subscription := bus.Subscribe("delivery.sent")
	defer subscription.Close()
	event := NewEvent("org-1", "delivery.sent", "delivery-1", "contact-1", nil)
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("publish event: %v", err)
	}
	select {
	case received := <-subscription.Events():
		if received.Type != "delivery.sent" {
			t.Fatalf("unexpected event type %q", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event was not delivered")
	}
}
