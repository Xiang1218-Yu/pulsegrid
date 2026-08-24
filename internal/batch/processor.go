package batch

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Item struct {
	ID      string         `json:"id"`
	Payload map[string]any `json:"payload,omitempty"`
}

type Result struct {
	ID         string    `json:"id"`
	Success    bool      `json:"success"`
	Attempts   int       `json:"attempts"`
	Error      string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

type Summary struct {
	Total       int           `json:"total"`
	Succeeded   int           `json:"succeeded"`
	Failed      int           `json:"failed"`
	Skipped     int           `json:"skipped"`
	Duration    time.Duration `json:"duration"`
	AverageTime time.Duration `json:"average_time"`
}

type Report struct {
	Summary Summary  `json:"summary"`
	Results []Result `json:"results"`
}

type Handler func(context.Context, Item) error

type Config struct {
	Workers  int
	Retry    int
	Continue bool
	Handler  Handler
}

type Processor struct {
	config Config
}

func New(config Config) *Processor {
	if config.Workers < 1 {
		config.Workers = runtime.GOMAXPROCS(0)
	}
	config.Workers = WorkerCount(config.Workers)
	if config.Retry < 0 {
		config.Retry = 0
	}
	return &Processor{config: config}
}

func (p *Processor) Run(ctx context.Context, items []Item) Report {
	started := time.Now()
	report := Report{Results: make([]Result, len(items))}
	if p.config.Handler == nil {
		for index, item := range items {
			report.Results[index] = failure(item, 0, errors.New("batch handler is not configured"))
		}
		report.Summary = summarize(report.Results, time.Since(started))
		return report
	}

	tasks := make(chan indexedItem)
	results := make(chan indexedItemResult)

	// stopCh is closed once when processing should halt early (either because a
	// handler failed in !Continue mode, or the context was cancelled). Closing a
	// channel is idempotent only when guarded, so sync.Once makes it safe for
	// several concurrent worker failures to signal stop without panicking.
	var stopOnce sync.Once
	stop := make(chan struct{})
	signalStop := func() { stopOnce.Do(func() { close(stop) }) }
	if ctxErr := ctx.Err(); ctxErr != nil {
		signalStop()
	}

	var workerGroup sync.WaitGroup
	for worker := 0; worker < p.config.Workers; worker++ {
		workerGroup.Add(1)
		go func() {
			defer workerGroup.Done()
			for task := range tasks {
				result := p.process(ctx, task.item)
				results <- indexedItemResult{index: task.index, result: result}
				if !result.Success && !p.config.Continue {
					signalStop()
					return
				}
			}
		}()
	}

	// The collector is the single writer of report.Results. Workers never touch
	// the slice; they report back through the results channel. This removes the
	// write race that existed when workers and the dispatch loop could both
	// store into the same slot (the dispatch loop's ctx.Done path used to
	// overwrite results that workers had already written).
	collectorDone := make(chan struct{})
	go func() {
		for item := range results {
			report.Results[item.index] = item.result
		}
		close(collectorDone)
	}()

	// Dispatch items until we run out, the context is cancelled, or a worker
	// signals an early stop. Dispatching happens here (single producer), so
	// there is no contention on the tasks channel.
	dispatched := 0
	for index, item := range items {
		select {
		case <-stop:
			// Remaining items are handled after the pool drains; stop now.
			goto drain
		case <-ctx.Done():
			signalStop()
			goto drain
		case tasks <- indexedItem{index: index, item: item}:
			dispatched++
		}
	}
drain:
	close(tasks)
	workerGroup.Wait()
	close(results)
	<-collectorDone

	// Fill any slots that were never dispatched. By this point the worker pool
	// has fully drained and the collector has exited, so no other goroutine is
	// touching report.Results — writing here is race-free.
	for index := dispatched; index < len(items); index++ {
		report.Results[index] = failure(items[index], 0, ctx.Err())
	}

	// summarize scans the now-fully-written slice, so the counts are always
	// consistent with the stored results (Succeeded+Failed+Skipped == Total).
	// The earlier stats counter only tallied processed items and could drift
	// below the real number under contention, which made monitoring undercount.
	report.Summary = summarize(report.Results, time.Since(started))
	return report
}

func (p *Processor) process(ctx context.Context, item Item) Result {
	started := time.Now()
	var lastErr error
	for attempt := 1; attempt <= p.config.Retry+1; attempt++ {
		if err := ctx.Err(); err != nil {
			return failureWithTimes(item, attempt-1, err, started, time.Now())
		}
		err := p.config.Handler(ctx, item)
		if err == nil {
			return Result{ID: item.ID, Success: true, Attempts: attempt, StartedAt: started, FinishedAt: time.Now()}
		}
		lastErr = err
		if !retryable(err) || attempt > p.config.Retry {
			break
		}
		delay := time.Duration(attempt*attempt) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return failureWithTimes(item, attempt, ctx.Err(), started, time.Now())
		case <-timer.C:
		}
	}
	return failureWithTimes(item, p.config.Retry+1, lastErr, started, time.Now())
}

func (p *Processor) Validate(items []Item) []error {
	result := make([]error, len(items))
	for index, item := range items {
		if item.ID == "" {
			result[index] = errors.New("item id is required")
			continue
		}
		if item.Payload == nil {
			result[index] = fmt.Errorf("item %s has nil payload", item.ID)
		}
	}
	return result
}

func Chunk(items []Item, size int) [][]Item {
	if size < 1 {
		size = 100
	}
	result := make([][]Item, 0)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		result = append(result, append([]Item(nil), items[start:end]...))
	}
	return result
}

func Merge(reports ...Report) Report {
	result := Report{}
	for _, report := range reports {
		result.Results = append(result.Results, report.Results...)
	}
	result.Summary = summarize(result.Results, 0)
	for _, report := range reports {
		result.Summary.Duration += report.Summary.Duration
	}
	if len(result.Results) > 0 {
		result.Summary.AverageTime = result.Summary.Duration / time.Duration(len(result.Results))
	}
	return result
}

func FilterFailed(report Report) []Result {
	result := make([]Result, 0)
	for _, item := range report.Results {
		if !item.Success {
			result = append(result, item)
		}
	}
	return result
}

func RetryReport(ctx context.Context, report Report, handler Handler) Report {
	items := make([]Item, 0)
	for _, result := range FilterFailed(report) {
		items = append(items, Item{ID: result.ID})
	}
	return New(Config{Workers: 1, Retry: 1, Continue: true, Handler: handler}).Run(ctx, items)
}

type indexedItem struct {
	index int
	item  Item
}

// indexedItemResult pairs a processed result with the original slice index so
// the collector goroutine can store it at the right position. Keeping the index
// out of Result (which has its own JSON shape) lets results travel back through
// a channel without the workers ever writing to the shared slice directly.
type indexedItemResult struct {
	index  int
	result Result
}

func failure(item Item, attempts int, err error) Result {
	now := time.Now()
	return failureWithTimes(item, attempts, err, now, now)
}

func failureWithTimes(item Item, attempts int, err error, started, finished time.Time) Result {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return Result{ID: item.ID, Success: false, Attempts: attempts, Error: message, StartedAt: started, FinishedAt: finished}
}

func summarize(results []Result, duration time.Duration) Summary {
	summary := Summary{Total: len(results), Duration: duration}
	for _, result := range results {
		if result.ID == "" {
			summary.Skipped++
			continue
		}
		if result.Success {
			summary.Succeeded++
		} else {
			summary.Failed++
		}
	}
	if summary.Total > 0 {
		summary.AverageTime = duration / time.Duration(summary.Total)
	}
	return summary
}

func retryable(err error) bool {
	if err == nil {
		return false
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}
