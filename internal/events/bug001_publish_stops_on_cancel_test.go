package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pulsegrid/internal/domain"
)

// Coverage markers: Bus.Publish, App.publish, Server.events.
func TestBug001PublishStopsOnCancel(t *testing.T) {
	bus := New(Config{Buffer: 1})
	sub := bus.Subscribe("campaign.started")
	defer sub.Close()
	event := domain.Event{Type: "campaign.started", ID: "first"}
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	ctx := &publishContext{done: make(chan struct{}), called: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- bus.Publish(ctx, domain.Event{Type: "campaign.started", ID: "second"}) }()
	<-ctx.called
	close(ctx.done)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("publish error = %v, want context canceled", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("publish did not stop after cancellation")
	}
}

type publishContext struct {
	done   chan struct{}
	called chan struct{}
	once   sync.Once
}

func (c *publishContext) Deadline() (time.Time, bool) { return time.Time{}, false }

func (c *publishContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.called) })
	return c.done
}

func (c *publishContext) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}

func (c *publishContext) Value(any) any { return nil }

func TestBug001PublishWithLiveContext(t *testing.T) {
	bus := New(Config{Buffer: 2})
	sub := bus.Subscribe("campaign.started")
	defer sub.Close()
	if err := bus.Publish(context.Background(), domain.Event{Type: "campaign.started", ID: "live"}); err != nil {
		t.Fatalf("live publish returned error: %v", err)
	}
}
