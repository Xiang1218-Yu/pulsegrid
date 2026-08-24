package batch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingHandler records every invocation and its success/failure outcome.
// It is safe for concurrent use, so it can validate that the reported summary
// matches what was actually processed.
func countingHandler(counts *atomic.Int64, succeed bool, ids map[string]bool, mu *sync.Mutex) Handler {
	return func(_ context.Context, item Item) error {
		counts.Add(1)
		mu.Lock()
		ids[item.ID] = true
		mu.Unlock()
		if succeed {
			return nil
		}
		return errors.New("boom")
	}
}

func TestRunConcurrentSummaryAccurate(t *testing.T) {
	// Under high concurrency the reported Succeeded count must equal the number
	// of items actually processed. The original implementation raced on the
	// stats counter and could overwrite summary counts that drifted below the
	// real number of completed items, making monitoring undercount successes.
	const itemCount = 500
	items := make([]Item, itemCount)
	for i := range items {
		items[i] = Item{ID: fmt.Sprintf("item-%d", i), Payload: map[string]any{"v": i}}
	}

	var processed atomic.Int64
	ids := make(map[string]bool)
	var mu sync.Mutex
	processor := New(Config{
		Workers:  16,
		Retry:    0,
		Continue: true,
		Handler:  countingHandler(&processed, true, ids, &mu),
	})

	report := processor.Run(context.Background(), items)

	if got := report.Summary.Succeeded; got != itemCount {
		t.Fatalf("succeeded = %d, want %d (processed %d)", got, itemCount, processed.Load())
	}
	if got := processed.Load(); got != itemCount {
		t.Fatalf("handler invoked %d times, want %d", got, itemCount)
	}
	if len(ids) != itemCount {
		t.Fatalf("distinct processed ids = %d, want %d", len(ids), itemCount)
	}
	if report.Summary.Failed != 0 || report.Summary.Skipped != 0 {
		t.Fatalf("failed=%d skipped=%d, want 0/0", report.Summary.Failed, report.Summary.Skipped)
	}
	if len(report.Results) != itemCount {
		t.Fatalf("results length = %d, want %d", len(report.Results), itemCount)
	}
	for i, r := range report.Results {
		if !r.Success || r.ID != items[i].ID {
			t.Fatalf("result[%d] = %+v, want success with id %q", i, r, items[i].ID)
		}
	}
}

func TestRunStopOnFailureLeavesRemainingUnprocessed(t *testing.T) {
	// In !Continue mode the first failure should stop dispatching. The already-
	// running handlers finish, but the never-dispatched tail must be reported as
	// failed (not silently skipped) and the handler must not be invoked for them.
	const itemCount = 100
	items := make([]Item, itemCount)
	for i := range items {
		items[i] = Item{ID: fmt.Sprintf("item-%d", i), Payload: map[string]any{"v": i}}
	}

	var processed atomic.Int64
	ids := make(map[string]bool)
	var mu sync.Mutex
	processor := New(Config{
		Workers:  8,
		Retry:    0,
		Continue: false,
		Handler:  countingHandler(&processed, false, ids, &mu),
	})

	report := processor.Run(context.Background(), items)

	if processed.Load() == 0 {
		t.Fatal("expected at least one handler invocation")
	}
	// Everything reported, nothing skipped, and Succeeded+Failed == Total.
	if report.Summary.Skipped != 0 {
		t.Fatalf("skipped = %d, want 0 (unprocessed tail must be reported as failed)", report.Summary.Skipped)
	}
	if got := report.Summary.Succeeded + report.Summary.Failed; got != itemCount {
		t.Fatalf("succeeded+failed = %d, want %d", got, itemCount)
	}
	if report.Summary.Succeeded != 0 {
		t.Fatalf("succeeded = %d, want 0 (handler always fails)", report.Summary.Succeeded)
	}
	if len(report.Results) != itemCount {
		t.Fatalf("results length = %d, want %d", len(report.Results), itemCount)
	}
	// No result should be left as a zero value: every slot must carry its id.
	for i, r := range report.Results {
		if r.ID != items[i].ID {
			t.Fatalf("result[%d].ID = %q, want %q", i, r.ID, items[i].ID)
		}
	}
	// The processed set must be a subset of all items (tail never ran).
	if len(ids) > itemCount {
		t.Fatalf("processed %d items, more than %d total", len(ids), itemCount)
	}
}

func TestRunSingleWorkerDoesNotHangOnFailure(t *testing.T) {
	// With a single worker and Continue=false, the original implementation
	// blocked forever on tasks <- ... after the worker returned, because nobody
	// was left to drain the unbuffered channel. This test fails (timeout) if
	// that hang regresses.
	const itemCount = 50
	items := make([]Item, itemCount)
	for i := range items {
		items[i] = Item{ID: fmt.Sprintf("item-%d", i), Payload: map[string]any{"v": i}}
	}
	var processed atomic.Int64
	ids := make(map[string]bool)
	var mu sync.Mutex
	processor := New(Config{
		Workers:  1,
		Retry:    0,
		Continue: false,
		Handler:  countingHandler(&processed, false, ids, &mu),
	})

	done := make(chan Report, 1)
	go func() { done <- processor.Run(context.Background(), items) }()
	select {
	case report := <-done:
		if len(report.Results) != itemCount {
			t.Fatalf("results length = %d, want %d", len(report.Results), itemCount)
		}
		if report.Summary.Skipped != 0 {
			t.Fatalf("skipped = %d, want 0", report.Summary.Skipped)
		}
		if got := report.Summary.Succeeded + report.Summary.Failed; got != itemCount {
			t.Fatalf("succeeded+failed = %d, want %d", got, itemCount)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run hung with a single worker and Continue=false")
	}
}

func TestRunSingleWorkerAllSucceed(t *testing.T) {
	const itemCount = 40
	items := make([]Item, itemCount)
	for i := range items {
		items[i] = Item{ID: fmt.Sprintf("item-%d", i), Payload: map[string]any{"v": i}}
	}
	var processed atomic.Int64
	ids := make(map[string]bool)
	var mu sync.Mutex
	processor := New(Config{
		Workers:  1,
		Retry:    0,
		Continue: true,
		Handler:  countingHandler(&processed, true, ids, &mu),
	})
	report := processor.Run(context.Background(), items)
	if report.Summary.Succeeded != itemCount {
		t.Fatalf("succeeded = %d, want %d", report.Summary.Succeeded, itemCount)
	}
	if processed.Load() != itemCount {
		t.Fatalf("processed = %d, want %d", processed.Load(), itemCount)
	}
}

func TestRunContextCancellation(t *testing.T) {
	// Cancelling the context mid-run must not race with worker result writes,
	// and every slot must still be populated.
	const itemCount = 200
	items := make([]Item, itemCount)
	for i := range items {
		items[i] = Item{ID: fmt.Sprintf("item-%d", i), Payload: map[string]any{"v": i}}
	}
	ctx, cancel := context.WithCancel(context.Background())

	gate := make(chan struct{})
	var processed atomic.Int64
	handler := func(_ context.Context, item Item) error {
		processed.Add(1)
		// Block the first few items so cancellation can happen during dispatch.
		<-gate
		return nil
	}
	processor := New(Config{Workers: 4, Retry: 0, Continue: true, Handler: handler})

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
		close(gate) // release any in-flight handlers so Run can finish.
	}()

	report := processor.Run(ctx, items)

	if len(report.Results) != itemCount {
		t.Fatalf("results length = %d, want %d", len(report.Results), itemCount)
	}
	// No zero-value results: every slot filled, and counts reconcile to Total.
	zero := 0
	for _, r := range report.Results {
		if r.ID == "" && !r.Success && r.Error == "" && r.Attempts == 0 {
			zero++
		}
	}
	if zero != 0 {
		t.Fatalf("found %d unfilled result slots", zero)
	}
	if got := report.Summary.Succeeded + report.Summary.Failed; got != itemCount {
		t.Fatalf("succeeded+failed = %d, want %d", got, itemCount)
	}
	if report.Summary.Skipped != 0 {
		t.Fatalf("skipped = %d, want 0", report.Summary.Skipped)
	}
	// Counted successes must equal actually processed-and-succeeded items.
	succeeded := 0
	for _, r := range report.Results {
		if r.Success {
			succeeded++
		}
	}
	if report.Summary.Succeeded != succeeded {
		t.Fatalf("summary succeeded = %d, recomputed = %d", report.Summary.Succeeded, succeeded)
	}
}

func TestRunRetryAndContinue(t *testing.T) {
	// A flaky handler that fails the first attempt per item, then succeeds.
	// With Retry>=1 and Continue=true everything should ultimately succeed and
	// the summary must reflect that.
	const itemCount = 30
	items := make([]Item, itemCount)
	for i := range items {
		items[i] = Item{ID: fmt.Sprintf("item-%d", i), Payload: map[string]any{"v": i}}
	}
	var mu sync.Mutex
	seen := make(map[string]int)
	handler := func(_ context.Context, item Item) error {
		mu.Lock()
		seen[item.ID]++
		attempts := seen[item.ID]
		mu.Unlock()
		if attempts < 2 {
			return errors.New("transient")
		}
		return nil
	}
	processor := New(Config{Workers: 4, Retry: 2, Continue: true, Handler: handler})
	report := processor.Run(context.Background(), items)
	if report.Summary.Succeeded != itemCount {
		t.Fatalf("succeeded = %d, want %d", report.Summary.Succeeded, itemCount)
	}
	if report.Summary.Failed != 0 {
		t.Fatalf("failed = %d, want 0", report.Summary.Failed)
	}
	for _, r := range report.Results {
		if r.Attempts < 2 {
			t.Fatalf("item %s attempts = %d, want >= 2", r.ID, r.Attempts)
		}
	}
}

func TestRunNoHandlerFailsAll(t *testing.T) {
	items := []Item{{ID: "a", Payload: map[string]any{}}, {ID: "b", Payload: map[string]any{}}}
	processor := New(Config{Workers: 2})
	report := processor.Run(context.Background(), items)
	if report.Summary.Failed != len(items) {
		t.Fatalf("failed = %d, want %d", report.Summary.Failed, len(items))
	}
	if report.Summary.Succeeded != 0 {
		t.Fatalf("succeeded = %d, want 0", report.Summary.Succeeded)
	}
}

func TestMergeAndChunk(t *testing.T) {
	items := make([]Item, 250)
	for i := range items {
		items[i] = Item{ID: fmt.Sprintf("item-%d", i), Payload: map[string]any{}}
	}
	chunks := Chunk(items, 100)
	if len(chunks) != 3 {
		t.Fatalf("chunks = %d, want 3", len(chunks))
	}
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	if total != len(items) {
		t.Fatalf("chunked total = %d, want %d", total, len(items))
	}

	merged := Merge()
	if merged.Summary.Total != 0 {
		t.Fatalf("empty merge total = %d, want 0", merged.Summary.Total)
	}
}
