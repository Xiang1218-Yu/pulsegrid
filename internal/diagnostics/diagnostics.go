package diagnostics

import (
	"runtime"
	"sync"
	"time"
)

type Registry struct {
	mu       sync.RWMutex
	started  time.Time
	counters map[string]int64
	gauges   map[string]float64
	healthy  map[string]bool
	messages map[string]string
}

type Snapshot struct {
	StartedAt     time.Time          `json:"started_at"`
	UptimeSeconds int64              `json:"uptime_seconds"`
	Counters      map[string]int64   `json:"counters"`
	Gauges        map[string]float64 `json:"gauges"`
	Healthy       map[string]bool    `json:"healthy"`
	Messages      map[string]string  `json:"messages"`
	Memory        runtime.MemStats   `json:"memory"`
}

func New() *Registry {
	return &Registry{started: time.Now().UTC(), counters: map[string]int64{}, gauges: map[string]float64{}, healthy: map[string]bool{}, messages: map[string]string{}}
}

func (r *Registry) Inc(name string, value int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] += value
}

func (r *Registry) SetGauge(name string, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = value
}

func (r *Registry) SetHealth(name string, healthy bool, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healthy[name] = healthy
	r.messages[name] = message
}

func (r *Registry) Healthy() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, healthy := range r.healthy {
		if !healthy {
			return false
		}
	}
	return true
}

func (r *Registry) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return Snapshot{
		StartedAt: r.started, UptimeSeconds: int64(time.Since(r.started).Seconds()),
		Counters: cloneInts(r.counters), Gauges: cloneFloats(r.gauges),
		Healthy: cloneBools(r.healthy), Messages: cloneMessages(r.messages), Memory: memory,
	}
}

func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters = map[string]int64{}
	r.gauges = map[string]float64{}
}

func cloneInts(source map[string]int64) map[string]int64 {
	out := map[string]int64{}
	for key, value := range source {
		out[key] = value
	}
	return out
}

func cloneFloats(source map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for key, value := range source {
		out[key] = value
	}
	return out
}

func cloneBools(source map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key, value := range source {
		out[key] = value
	}
	return out
}

func cloneMessages(source map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range source {
		out[key] = value
	}
	return out
}
