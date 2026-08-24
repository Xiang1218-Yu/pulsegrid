package jobs

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"log/slog"
)

// quietLogger discards output so the test log stays readable.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
}

// newStartedQueue builds a queue with one worker and starts it.
func newStartedQueue(t *testing.T, capacity int) *Queue {
	t.Helper()
	q := New(Config{Workers: 1, Capacity: capacity, Logger: quietLogger()})
	q.Start()
	t.Cleanup(q.Stop)
	return q
}

// TestExecuteRecoversHandlerPanic verifies that a panicking handler does not
// kill the worker goroutine or the process: the panicked job is counted as
// failed+panicked, and a subsequent job on the same worker still completes.
func TestExecuteRecoversHandlerPanic(t *testing.T) {
	q := newStartedQueue(t, 8)
	var ran atomic.Int64

	q.Register("boom", func(_ context.Context, _ Job) error {
		panic("kaboom")
	})
	q.Register("ok", func(_ context.Context, _ Job) error {
		ran.Add(1)
		return nil
	})

	if err := q.Enqueue(context.Background(), Job{Type: "boom", MaxRetry: 0}); err != nil {
		t.Fatalf("enqueue boom: %v", err)
	}
	if err := q.Enqueue(context.Background(), Job{Type: "ok", MaxRetry: 0}); err != nil {
		t.Fatalf("enqueue ok: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		stats := q.Stats()
		if ran.Load() == 1 && stats.Panicked == 1 && stats.Failed >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout: ran=%d stats=%+v", ran.Load(), stats)
		default:
		}
		time.Sleep(5 * time.Millisecond)
	}

	stats := q.Stats()
	if stats.Panicked != 1 {
		t.Fatalf("expected 1 panicked, got %d", stats.Panicked)
	}
	if stats.Finished < 1 {
		t.Fatalf("expected at least 1 finished, got %d", stats.Finished)
	}
}

// TestPanicIsNotRetried documents the deliberate semantics: a panicking
// handler is a fatal, non-transient failure for that job, so it is recovered
// once and not retried (retrying identical input would just panic again).
// The worker still survives to run subsequent jobs.
func TestPanicIsNotRetried(t *testing.T) {
	q := newStartedQueue(t, 8)
	var attempts atomic.Int64

	q.Register("boom", func(_ context.Context, _ Job) error {
		attempts.Add(1)
		panic("deterministic fault")
	})
	q.Register("next", func(_ context.Context, _ Job) error { return nil })

	if err := q.Enqueue(context.Background(), Job{Type: "boom", MaxRetry: 3}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Enqueue(context.Background(), Job{Type: "next", MaxRetry: 0}); err != nil {
		t.Fatalf("enqueue next: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		stats := q.Stats()
		if stats.Finished >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout: attempts=%d stats=%+v", attempts.Load(), stats)
		default:
		}
		time.Sleep(5 * time.Millisecond)
	}

	stats := q.Stats()
	if attempts.Load() != 1 {
		t.Fatalf("panicking job should not be retried, attempts=%d", attempts.Load())
	}
	if stats.Panicked != 1 {
		t.Fatalf("expected 1 panicked, got %d", stats.Panicked)
	}
	if stats.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", stats.Failed)
	}
	if stats.Finished != 1 {
		t.Fatalf("expected 1 finished (the follow-up job), got %d", stats.Finished)
	}
}

// TestBackoffHonoursStop verifies the retry backoff gives up when the queue is
// stopping instead of blocking shutdown.
func TestBackoffHonoursStop(t *testing.T) {
	q := New(Config{Workers: 1, Capacity: 8, Logger: quietLogger()})
	q.Start()

	started := make(chan struct{})
	q.Register("slow", func(_ context.Context, job Job) error {
		close(started)
		return errors.New("always fails")
	})

	_ = q.Enqueue(context.Background(), Job{Type: "slow", MaxRetry: 5})
	<-started

	// Stop during the backoff window — this must return promptly.
	stopDone := make(chan struct{})
	go func() {
		q.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop blocked during backoff")
	}
}

// TestEnqueueAfterStop returns a deterministic error, not a panic, once the
// queue is shutting down.
func TestEnqueueAfterStop(t *testing.T) {
	q := New(Config{Workers: 1, Capacity: 8, Logger: quietLogger()})
	q.Start()
	q.Stop()

	if err := q.Enqueue(context.Background(), Job{Type: "ok"}); err == nil {
		t.Fatal("expected error enqueuing after stop")
	}
}

// TestConcurrentPanickingHandlers runs many panicking jobs across several
// workers to make sure all of them are recovered and counted, and that the
// queue stays drainable afterwards.
func TestConcurrentPanickingHandlers(t *testing.T) {
	q := New(Config{Workers: 4, Capacity: 256, Logger: quietLogger()})
	q.Start()
	t.Cleanup(q.Stop)

	q.Register("panic", func(_ context.Context, _ Job) error { panic("nope") })
	q.Register("fine", func(_ context.Context, _ Job) error { return nil })

	const panics = 40
	const fines = 40
	for i := 0; i < panics; i++ {
		if err := q.Enqueue(context.Background(), Job{Type: "panic", MaxRetry: 0}); err != nil {
			t.Fatalf("enqueue panic %d: %v", i, err)
		}
	}
	for i := 0; i < fines; i++ {
		if err := q.Enqueue(context.Background(), Job{Type: "fine", MaxRetry: 0}); err != nil {
			t.Fatalf("enqueue fine %d: %v", i, err)
		}
	}

	deadline := time.After(3 * time.Second)
	for {
		stats := q.Stats()
		if stats.Panicked == panics && stats.Finished == fines {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout: stats=%+v", stats)
		default:
		}
		time.Sleep(5 * time.Millisecond)
	}
}
