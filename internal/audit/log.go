package audit

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type Action string

const (
	Create  Action = "create"
	Update  Action = "update"
	Delete  Action = "delete"
	Archive Action = "archive"
	Restore Action = "restore"
	Send    Action = "send"
	Open    Action = "open"
	Click   Action = "click"
	Import  Action = "import"
	Export  Action = "export"
	System  Action = "system"
)

type Entry struct {
	ID             string            `json:"id"`
	OrganizationID string            `json:"organization_id,omitempty"`
	Actor          string            `json:"actor,omitempty"`
	Action         Action            `json:"action"`
	Resource       string            `json:"resource"`
	ResourceID     string            `json:"resource_id,omitempty"`
	Message        string            `json:"message"`
	Changes        map[string]string `json:"changes,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	OccurredAt     time.Time         `json:"occurred_at"`
}

type Filter struct {
	OrganizationID string
	Actor          string
	Action         Action
	Resource       string
	ResourceID     string
	Search         string
	From           time.Time
	To             time.Time
	Limit          int
}

type Summary struct {
	Total      int            `json:"total"`
	ByAction   map[string]int `json:"by_action"`
	ByResource map[string]int `json:"by_resource"`
	Latest     time.Time      `json:"latest"`
}

type Log struct {
	mu       sync.RWMutex
	limit    int
	sequence uint64
	entries  []Entry
}

func New(limit int) *Log {
	if limit < 1 {
		limit = 10000
	}
	return &Log{limit: limit, entries: []Entry{}}
}

func (l *Log) Append(entry Entry) Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sequence++
	if entry.ID == "" {
		entry.ID = "audit-" + formatID(l.sequence)
	}
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = time.Now().UTC()
	}
	entry.Changes = cloneStrings(entry.Changes)
	entry.Metadata = cloneStrings(entry.Metadata)
	l.entries = append(l.entries, entry)
	if len(l.entries) > l.limit {
		offset := len(l.entries) - l.limit
		l.entries = append([]Entry(nil), l.entries[offset:]...)
	}
	return clone(entry)
}

func (l *Log) Record(organizationID, actor string, action Action, resource, resourceID, message string, changes, metadata map[string]string) Entry {
	return l.Append(Entry{
		OrganizationID: organizationID,
		Actor:          actor,
		Action:         action,
		Resource:       resource,
		ResourceID:     resourceID,
		Message:        message,
		Changes:        changes,
		Metadata:       metadata,
		OccurredAt:     time.Now().UTC(),
	})
}

func (l *Log) List(filter Filter) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	result := make([]Entry, 0)
	for _, entry := range l.entries {
		if filter.OrganizationID != "" && entry.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.Actor != "" && entry.Actor != filter.Actor {
			continue
		}
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.Resource != "" && entry.Resource != filter.Resource {
			continue
		}
		if filter.ResourceID != "" && entry.ResourceID != filter.ResourceID {
			continue
		}
		if !filter.From.IsZero() && entry.OccurredAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && entry.OccurredAt.After(filter.To) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(entry.Message+" "+entry.Resource+" "+entry.Actor), search) {
			continue
		}
		result = append(result, clone(entry))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OccurredAt.After(result[j].OccurredAt) })
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result
}

func (l *Log) Summary(filter Filter) Summary {
	rows := l.List(filter)
	result := Summary{Total: len(rows), ByAction: map[string]int{}, ByResource: map[string]int{}}
	for _, entry := range rows {
		result.ByAction[string(entry.Action)]++
		result.ByResource[entry.Resource]++
		if entry.OccurredAt.After(result.Latest) {
			result.Latest = entry.OccurredAt
		}
	}
	return result
}

func (l *Log) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

func (l *Log) Since(value time.Time) []Entry {
	return l.List(Filter{From: value, Limit: 100000})
}

func (l *Log) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = nil
}

func clone(entry Entry) Entry {
	entry.Changes = cloneStrings(entry.Changes)
	entry.Metadata = cloneStrings(entry.Metadata)
	return entry
}

func cloneStrings(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	out := map[string]string{}
	for key, value := range source {
		out[key] = value
	}
	return out
}

func formatID(value uint64) string {
	return time.Unix(int64(value), 0).UTC().Format("20060102150405")
}
