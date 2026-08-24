package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
)

// streamingWriter is a minimal http.ResponseWriter that supports streaming by
// exposing every write and honoring http.Flusher. It lets us drive the events
// handler deterministically without a real network round trip.
type streamingWriter struct {
	mu       sync.Mutex
	header   http.Header
	status   int
	body     strings.Builder
	flushed  chan struct{}
	hijacked bool
}

func newStreamingWriter() *streamingWriter {
	return &streamingWriter{header: http.Header{}, flushed: make(chan struct{}, 1)}
}

func (w *streamingWriter) Header() http.Header { return w.header }
func (w *streamingWriter) WriteHeader(code int) { w.status = code }
func (w *streamingWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(b)
}
func (w *streamingWriter) Flush() {
	select {
	case w.flushed <- struct{}{}:
	default:
	}
}

// readJSONEvents decodes the buffered ndjson body into events.
func (w *streamingWriter) events() []domain.Event {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []domain.Event
	dec := json.NewDecoder(strings.NewReader(w.body.String()))
	for dec.More() {
		var ev domain.Event
		if err := dec.Decode(&ev); err != nil {
			break
		}
		out = append(out, ev)
	}
	return out
}

// A client whose request context is canceled (disconnect) must release its
// subscription so the bus does not leak dead subscribers or route into their
// orphaned buffers.
func TestEventsHandlerReleasesSubscriptionOnContextCancel(t *testing.T) {
	bus := events.New(events.Config{Buffer: 8, DropWhenBusy: false})
	defer bus.Close()
	server := New(Config{Events: bus})

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/v1/events?topic=topic", nil)
	w := newStreamingWriter()

	done := make(chan struct{})
	go func() {
		defer close(done)
		server.Handler().ServeHTTP(w, req)
	}()

	// Wait until the handler has registered its subscription.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && bus.SubscriberCount() == 0 {
		time.Sleep(2 * time.Millisecond)
	}
	if bus.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after connect, got %d", bus.SubscriberCount())
	}

	// Simulate the client dropping the connection.
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after request cancellation")
	}

	// The deferred Close must have removed the subscription.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && bus.SubscriberCount() != 0 {
		time.Sleep(2 * time.Millisecond)
	}
	if got := bus.SubscriberCount(); got != 0 {
		t.Fatalf("expected 0 subscribers after disconnect, got %d", got)
	}
}

// While a stream is active it must still receive every published event, even
// as other subscriptions are created and torn down around it.
func TestEventsHandlerLiveStreamReceivesDuringChurn(t *testing.T) {
	bus := events.New(events.Config{Buffer: 256, DropWhenBusy: false})
	defer bus.Close()
	server := New(Config{Events: bus})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/v1/events?topic=topic", nil)
	w := newStreamingWriter()

	go server.Handler().ServeHTTP(w, req)

	// Wait for registration.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && bus.SubscriberCount() == 0 {
		time.Sleep(2 * time.Millisecond)
	}

	const total = 300
	for n := 0; n < total; n++ {
		if n%5 == 0 {
			// Create and immediately tear down another subscription to mimic
			// churn; its removal must not disturb the live stream.
			bus.Subscribe("topic").Close()
		}
		_ = bus.Publish(context.Background(), domain.Event{Type: "topic", ID: "e"})
	}

	// Wait until the live stream has flushed all events.
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(w.events()) >= total {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if got := len(w.events()); got != total {
		t.Fatalf("live stream received %d/%d events", got, total)
	}
}
