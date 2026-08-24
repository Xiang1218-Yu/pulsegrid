package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"pulsegrid/internal/domain"
)

type Handler func(context.Context, Job) error

type Job struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload,omitempty"`
	MaxRetry  int            `json:"max_retry"`
	Attempt   int            `json:"attempt"`
	CreatedAt time.Time      `json:"created_at"`
}

type Config struct {
	Workers  int
	Capacity int
	Logger   *slog.Logger
}

type Queue struct {
	config   Config
	mu       sync.RWMutex
	handlers map[string]Handler
	items    chan Job
	stop     chan struct{}
	done     chan struct{}
	started  atomic.Bool
	closed   atomic.Bool
	wg       sync.WaitGroup
	accepted atomic.Int64
	finished atomic.Int64
	failed   atomic.Int64
}

type Stats struct {
	Depth    int   `json:"depth"`
	Workers  int   `json:"workers"`
	Accepted int64 `json:"accepted"`
	Finished int64 `json:"finished"`
	Failed   int64 `json:"failed"`
}

var ErrNotStarted = errors.New("queue is not started")

func New(config Config) *Queue {
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.Capacity < 1 {
		config.Capacity = config.Workers * 4
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	return &Queue{config: config, handlers: map[string]Handler{}, items: make(chan Job, config.Capacity), stop: make(chan struct{}), done: make(chan struct{})}
}

func (q *Queue) Register(kind string, handler Handler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[kind] = handler
}

func (q *Queue) Start() {
	if !q.started.CompareAndSwap(false, true) {
		return
	}
	for index := 0; index < q.config.Workers; index++ {
		q.wg.Add(1)
		go q.worker(index + 1)
	}
}

func (q *Queue) Stop() {
	if !q.started.Load() || !q.closed.CompareAndSwap(false, true) {
		return
	}
	close(q.stop)
	q.wg.Wait()
	close(q.done)
}

func (q *Queue) Enqueue(ctx context.Context, job Job) error {
	if !q.started.Load() {
		return ErrNotStarted
	}
	if q.closed.Load() {
		return domain.ErrInvalidState
	}
	if job.ID == "" {
		job.ID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-q.stop:
		return domain.ErrInvalidState
	case q.items <- job:
		q.accepted.Add(1)
		return nil
	default:
		return domain.ErrConflict
	}
}

func (q *Queue) Stats() Stats {
	return Stats{Depth: len(q.items), Workers: q.config.Workers, Accepted: q.accepted.Load(), Finished: q.finished.Load(), Failed: q.failed.Load()}
}

func (q *Queue) worker(index int) {
	defer q.wg.Done()
	name := fmt.Sprintf("worker-%d", index)
	for {
		select {
		case <-q.stop:
			return
		case job := <-q.items:
			q.execute(name, job)
		}
	}
}

func (q *Queue) execute(worker string, job Job) {
	handler, ok := q.handler(job.Type)
	if !ok {
		q.failed.Add(1)
		q.config.Logger.Warn("missing job handler", "type", job.Type)
		return
	}
	maxRetry := job.MaxRetry
	if maxRetry < 0 {
		maxRetry = 0
	}
	for attempt := 0; attempt <= maxRetry; attempt++ {
		job.Attempt = attempt + 1
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := handler(ctx, job)
		cancel()
		if err == nil {
			q.finished.Add(1)
			q.config.Logger.Debug("job finished", "job_id", job.ID, "worker", worker)
			return
		}
		if attempt == maxRetry {
			q.failed.Add(1)
			q.config.Logger.Error("job failed", "job_id", job.ID, "error", err)
			return
		}
		time.Sleep(time.Duration(5*(1<<min(attempt, 5))) * time.Millisecond)
	}
}

func (q *Queue) handler(kind string) (Handler, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	handler, ok := q.handlers[kind]
	return handler, ok
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
