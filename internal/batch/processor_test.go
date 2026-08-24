package batch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// testRuns is the fixed number of iterations every consistency check runs for,
// matching the required race-checked verification cadence.
const testRuns = 20

// cyclicBarrier is a standard-library cyclic barrier. It forces a fixed number
// of goroutines to rendezvous before any of them may proceed, then resets. It
// is the "synchronization barrier" that stabilises reproduction of the
// parallel-write window: every batch of workers enters process() together and
// releases together, so concurrent record writes overlap deterministically
// rather than by scheduler luck. With the per-index-write fix there is no race
// to find; with the previous shared-append code this barrier made the
// lost/duplicated-record failure reproducible on every run.
type cyclicBarrier struct {
	n     int
	count int
	mu    sync.Mutex
	cond  *sync.Cond
	gen   int
}

func newCyclicBarrier(n int) *cyclicBarrier {
	b := &cyclicBarrier{n: n}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *cyclicBarrier) wait() {
	b.mu.Lock()
	defer b.mu.Unlock()
	gen := b.gen
	b.count++
	if b.count == b.n {
		b.count = 0
		b.gen++
		b.cond.Broadcast()
		return
	}
	for gen == b.gen {
		b.cond.Wait()
	}
}

// TestProcessor_ParallelConsistency runs the processor a fixed number of times
// under -race with a barrier that guarantees worker overlap, and asserts that
// no records are lost or duplicated and that the summary counts exactly match
// the actual records. Run with: go test ./internal/batch/ -race -count=1
func TestProcessor_ParallelConsistency(t *testing.T) {
	const (
		workers = 8
		// items is a multiple of workers so every barrier generation is full
		// and the cyclic barrier never deadlocks on a partial batch.
		items = 40
	)

	for run := 0; run < testRuns; run++ {
		barrier := newCyclicBarrier(workers)
		input := make([]Item, 0, items)
		for i := 0; i < items; i++ {
			input = append(input, Item{
				ID:      fmt.Sprintf("item-%03d", i),
				Payload: map[string]any{"index": i},
			})
		}

		// Half the items fail on purpose so success/failure accounting is
		// actually exercised, not just the all-success happy path.
		handler := func(ctx context.Context, item Item) error {
			barrier.wait() // every batch of workers enters together
			idx, _ := item.Payload["index"].(int)
			if idx%2 == 1 {
				return errors.New("simulated failure")
			}
			return nil
		}

		proc := New(Config{Workers: workers, Retry: 0, Continue: true, Handler: handler})
		report := proc.Run(context.Background(), input)

		// 1. No records lost or duplicated: length must equal input length.
		if got := len(report.Results); got != items {
			t.Fatalf("run %d: expected %d results, got %d", run, items, got)
		}

		// 2. Every input id present exactly once.
		seen := make(map[string]int, items)
		for _, r := range report.Results {
			seen[r.ID]++
		}
		for id, c := range seen {
			if c != 1 {
				t.Fatalf("run %d: id %s recorded %d times (duplicate or missing)", run, id, c)
			}
		}

		// 3. Summary counts agree with the actual records.
		var succ, fail, skip int
		for _, r := range report.Results {
			switch {
			case r.ID == "":
				skip++
			case r.Success:
				succ++
			default:
				fail++
			}
		}
		if report.Summary.Succeeded != succ {
			t.Fatalf("run %d: succeeded summary %d != actual %d", run, report.Summary.Succeeded, succ)
		}
		if report.Summary.Failed != fail {
			t.Fatalf("run %d: failed summary %d != actual %d", run, report.Summary.Failed, fail)
		}
		if report.Summary.Skipped != skip {
			t.Fatalf("run %d: skipped summary %d != actual %d", run, report.Summary.Skipped, skip)
		}
		if got := report.Summary.Succeeded + report.Summary.Failed + report.Summary.Skipped; got != items {
			t.Fatalf("run %d: summary total %d != %d", run, got, items)
		}
		if report.Summary.Total != items {
			t.Fatalf("run %d: summary.Total %d != %d", run, report.Summary.Total, items)
		}
	}
}

// TestProcessor_DispatchCancellationAccounting verifies that when dispatch is
// interrupted (context cancelled), every unstarted item still gets a record and
// the summary still reconciles. This guards the same class of count-vs-records
// drift for the cancellation path.
func TestProcessor_DispatchCancellationAccounting(t *testing.T) {
	const items = 50

	started := make(chan struct{}, 1)
	block := make(chan struct{})
	handler := func(ctx context.Context, item Item) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-block // hold the first worker so dispatch cannot complete normally
		return nil
	}
	proc := New(Config{Workers: 1, Retry: 0, Continue: true, Handler: handler})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel() // cancel once a worker is mid-flight
		close(block)
	}()

	input := make([]Item, 0, items)
	for i := 0; i < items; i++ {
		input = append(input, Item{ID: fmt.Sprintf("x-%d", i)})
	}
	report := proc.Run(ctx, input)

	if got := len(report.Results); got != items {
		t.Fatalf("expected %d results, got %d", items, got)
	}
	var succ, fail int
	for _, r := range report.Results {
		if r.Success {
			succ++
		} else {
			fail++
		}
	}
	if report.Summary.Succeeded != succ || report.Summary.Failed != fail {
		t.Fatalf("summary (%d/%d) != actual (%d/%d)", report.Summary.Succeeded, report.Summary.Failed, succ, fail)
	}
	if report.Summary.Succeeded+report.Summary.Failed != items {
		t.Fatalf("summary total %d != %d", report.Summary.Succeeded+report.Summary.Failed, items)
	}
}

// TestProcessor_ContinueStopsOnFailure confirms that with Continue disabled, a
// failure halts further processing without losing records already produced and
// without deadlocking dispatch (the previous code could hang on an unbuffered
// tasks channel once workers stopped draining it).
func TestProcessor_ContinueStopsOnFailure(t *testing.T) {
	const items = 30
	processed := make(chan string, items)
	handler := func(ctx context.Context, item Item) error {
		processed <- item.ID
		return errors.New("stop")
	}
	proc := New(Config{Workers: 2, Retry: 0, Continue: false, Handler: handler})

	input := make([]Item, 0, items)
	for i := 0; i < items; i++ {
		input = append(input, Item{ID: fmt.Sprintf("s-%d", i)})
	}

	done := make(chan Report, 1)
	go func() { done <- proc.Run(context.Background(), input) }()

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("Run deadlocked: Continue=false did not unblock dispatch")
	case report := <-done:
		// Every position must carry a record: some processed, the rest
		// cancelled because the run context was cancelled on first failure.
		if got := len(report.Results); got != items {
			t.Fatalf("expected %d results, got %d", items, got)
		}
		// Every input id present exactly once (avoids lexical-order traps:
		// "s-9" sorts after "s-29", so counting is the robust check).
		seen := make(map[string]int, items)
		for _, r := range report.Results {
			seen[r.ID]++
		}
		if len(seen) != items {
			t.Fatalf("expected %d distinct ids, got %d", items, len(seen))
		}
		for i := 0; i < items; i++ {
			id := fmt.Sprintf("s-%d", i)
			if seen[id] != 1 {
				t.Fatalf("id %s recorded %d times", id, seen[id])
			}
		}
		// Summary must reconcile with the records.
		var succ, fail int
		for _, r := range report.Results {
			if r.Success {
				succ++
			} else {
				fail++
			}
		}
		if succ != 0 {
			t.Fatalf("Continue=false should have stopped all successes, got %d", succ)
		}
		if report.Summary.Failed != fail || report.Summary.Succeeded != succ {
			t.Fatalf("summary (%d/%d) != actual (%d/%d)", report.Summary.Succeeded, report.Summary.Failed, succ, fail)
		}
		if report.Summary.Succeeded+report.Summary.Failed != items {
			t.Fatalf("summary total %d != %d", report.Summary.Succeeded+report.Summary.Failed, items)
		}
	}
}
