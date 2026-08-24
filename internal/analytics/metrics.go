package analytics

import (
	"sort"
	"sync"
	"time"

	"pulsegrid/internal/domain"
)

type Store struct {
	mu      sync.RWMutex
	metrics []domain.Metric
}

type Filter struct {
	OrganizationID string
	Name           string
	From           time.Time
	To             time.Time
	Limit          int
}

type Summary struct {
	Name    string  `json:"name"`
	Count   int     `json:"count"`
	Sum     float64 `json:"sum"`
	Average float64 `json:"average"`
	Minimum float64 `json:"minimum"`
	Maximum float64 `json:"maximum"`
}

type Daily struct {
	Day   string  `json:"day"`
	Count int     `json:"count"`
	Value float64 `json:"value"`
}

func New() *Store { return &Store{metrics: []domain.Metric{}} }

func (s *Store) Record(metric domain.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if metric.RecordedAt.IsZero() {
		metric.RecordedAt = time.Now().UTC()
	}
	s.metrics = append(s.metrics, clone(metric))
}

func (s *Store) RecordValue(organizationID, name string, value float64, dimensions map[string]string, at time.Time) domain.Metric {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	metric := domain.Metric{ID: "metric-" + at.Format("20060102T150405.000000000"), OrganizationID: organizationID, Name: name, Value: value, Dimensions: cloneStrings(dimensions), RecordedAt: at}
	s.Record(metric)
	return metric
}

func (s *Store) List(filter Filter) []domain.Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Metric, 0)
	for _, metric := range s.metrics {
		if filter.OrganizationID != "" && metric.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.Name != "" && metric.Name != filter.Name {
			continue
		}
		if !filter.From.IsZero() && metric.RecordedAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && metric.RecordedAt.After(filter.To) {
			continue
		}
		result = append(result, clone(metric))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RecordedAt.Before(result[j].RecordedAt) })
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[len(result)-filter.Limit:]
	}
	return result
}

func (s *Store) Summarize(filter Filter) []Summary {
	rows := s.List(filter)
	groups := map[string][]domain.Metric{}
	for _, row := range rows {
		groups[row.Name] = append(groups[row.Name], row)
	}
	result := make([]Summary, 0, len(groups))
	for name, values := range groups {
		summary := Summary{Name: name, Count: len(values)}
		if len(values) > 0 {
			summary.Minimum, summary.Maximum = values[0].Value, values[0].Value
		}
		for _, value := range values {
			summary.Sum += value.Value
			if value.Value < summary.Minimum {
				summary.Minimum = value.Value
			}
			if value.Value > summary.Maximum {
				summary.Maximum = value.Value
			}
		}
		if summary.Count > 0 {
			summary.Average = summary.Sum / float64(summary.Count)
		}
		result = append(result, summary)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (s *Store) Daily(filter Filter) []Daily {
	rows := s.List(filter)
	groups := map[string]*Daily{}
	for _, row := range rows {
		day := row.RecordedAt.UTC().Format("2006-01-02")
		if groups[day] == nil {
			groups[day] = &Daily{Day: day}
		}
		groups[day].Count++
		groups[day].Value += row.Value
	}
	result := make([]Daily, 0, len(groups))
	for _, value := range groups {
		result = append(result, *value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Day < result[j].Day })
	return result
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.metrics)
}

func clone(value domain.Metric) domain.Metric {
	value.Dimensions = cloneStrings(value.Dimensions)
	return value
}
func cloneStrings(source map[string]string) map[string]string {
	if source == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for key, value := range source {
		out[key] = value
	}
	return out
}
