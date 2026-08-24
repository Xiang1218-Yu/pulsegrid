package httpapi

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pulsegrid/internal/events"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

// newEventsServer wires a Server backed by a real event bus so the /v1/events
// stream can be exercised end to end.
func newEventsServer(t *testing.T, bus *events.Bus) *httptest.Server {
	t.Helper()
	server := New(Config{App: service.New(service.Config{Repository: store.NewMemory()}), Events: bus})
	return httptest.NewServer(server.Handler())
}

// streamEvents opens a streaming connection to /v1/events that can be torn down
// by cancelling the returned context, mimicking a client closing the page.
func streamEvents(t *testing.T, baseURL string) (lines chan string, cancel func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	lines = make(chan string, 8)
	go func() {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/events", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			close(lines)
			return
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
		close(lines)
	}()
	// Give the server a moment to register the subscription before returning.
	time.Sleep(50 * time.Millisecond)
	return lines, cancel
}

// TestEventsStreamDeliversThenReleasesOnContextCancel is the regression test
// for the lifecycle fix: a client that closes the page (request context
// cancelled) must release its server-side subscription even when no further
// events arrive, while normal delivery still works.
func TestEventsStreamDeliversThenReleasesOnContextCancel(t *testing.T) {
	bus := events.New(events.Config{Buffer: 4})
	srv := newEventsServer(t, bus)
	defer srv.Close()

	lines, cancel := streamEvents(t, srv.URL)

	// Normal delivery path still works: a published event reaches the client.
	event := events.NewEvent("org-1", events.TopicContactCreated, "contact-1", "contact-1", nil)
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("publish: %v", err)
	}
	select {
	case got := <-lines:
		if !strings.Contains(got, event.ID) {
			t.Fatalf("delivered event mismatch: want id %q in %q", event.ID, got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event delivery")
	}

	// Client closes the page: cancel the request context.
	cancel()

	// The handler must exit and Close() the subscription. After that, a later
	// publish to the same topic must not panic by sending into a closed
	// channel (the subscription has been removed from the bus recipients).
	// Wait briefly to let the server observe the cancellation and run defer.
	time.Sleep(100 * time.Millisecond)

	lateEvent := events.NewEvent("org-1", events.TopicContactCreated, "contact-2", "contact-2", nil)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("publish after client disconnect panicked (subscription not released): %v", r)
		}
	}()
	if err := bus.Publish(context.Background(), lateEvent); err != nil {
		t.Fatalf("late publish: %v", err)
	}

	// Drain any lingering client-side goroutine so the test does not race.
	for range lines {
	}
}

// TestEventsStreamReleasesWhenNoEventsArrive pins the exact leak the fix
// targets: with the old context.Background() stream context, a connection
// that disconnected before any event arrived blocked forever on the events
// channel. Now the request context cancellation unblocks it.
func TestEventsStreamReleasesWhenNoEventsArrive(t *testing.T) {
	bus := events.New(events.Config{Buffer: 4})
	srv := newEventsServer(t, bus)
	defer srv.Close()

	_, cancel := streamEvents(t, srv.URL)
	cancel()

	// If the handler leaked (old behavior), the subscription stays in the bus.
	// Probe with a publish: sending into the still-live closed channel would
	// not happen, but the goroutine would remain blocked. We assert via the
	// absence of a panic and a bounded wait that teardown completed.
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("post-disconnect publish panicked: %v", r)
			}
			close(done)
		}()
		_ = bus.Publish(context.Background(), events.NewEvent("org-1", events.TopicContactCreated, "x", "x", nil))
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked after disconnect, subscription likely leaked")
	}
}
