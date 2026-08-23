package pagination

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	Page       int    `json:"page"`
	Size       int    `json:"size"`
	Offset     int    `json:"-"`
	Sort       string `json:"sort,omitempty"`
	Descending bool   `json:"descending"`
}

type Result[T any] struct {
	Items       []T  `json:"items"`
	Page        int  `json:"page"`
	Size        int  `json:"size"`
	Total       int  `json:"total"`
	Pages       int  `json:"pages"`
	HasNext     bool `json:"has_next"`
	HasPrevious bool `json:"has_previous"`
}

type Cursor struct {
	Value string `json:"value,omitempty"`
	Limit int    `json:"limit"`
}

type Window struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func Parse(values url.Values) Request {
	page := positive(values.Get("page"), 1)
	size := positive(values.Get("size"), 25)
	if size > 500 {
		size = 500
	}
	rawSort := strings.TrimSpace(values.Get("sort"))
	descending := strings.HasPrefix(rawSort, "-")
	rawSort = strings.TrimPrefix(rawSort, "-")
	return Request{Page: page, Size: size, Offset: (page - 1) * size, Sort: rawSort, Descending: descending}
}

func Apply[T any](items []T, request Request) Result[T] {
	if request.Page < 1 {
		request.Page = 1
	}
	if request.Size < 1 {
		request.Size = 25
	}
	if request.Offset < 0 {
		request.Offset = 0
	}
	start := request.Offset
	if start > len(items) {
		start = len(items)
	}
	end := start + request.Size
	if end > len(items) {
		end = len(items)
	}
	pages := 0
	if len(items) > 0 {
		pages = (len(items) + request.Size - 1) / request.Size
	}
	return Result[T]{
		Items:       items[start:end],
		Page:        request.Page,
		Size:        request.Size,
		Total:       len(items),
		Pages:       pages,
		HasNext:     end < len(items),
		HasPrevious: start > 0,
	}
}

func ApplyCursor[T any](items []T, cursor Cursor, key func(T) string) (Result[T], Cursor) {
	if cursor.Limit < 1 {
		cursor.Limit = 25
	}
	start := 0
	if cursor.Value != "" {
		for index, item := range items {
			if key(item) == cursor.Value {
				start = index + 1
				break
			}
		}
	}
	end := start + cursor.Limit
	if end > len(items) {
		end = len(items)
	}
	next := Cursor{}
	if end < len(items) && end > 0 {
		next.Value = key(items[end-1])
		next.Limit = cursor.Limit
	}
	return Result[T]{Items: items[start:end], Page: 1, Size: cursor.Limit, Total: len(items), Pages: 1, HasNext: end < len(items), HasPrevious: start > 0}, next
}

func ParseWindow(values url.Values, now time.Time) Window {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	from := parseTime(values.Get("from"), now.AddDate(0, 0, -7))
	to := parseTime(values.Get("to"), now)
	if to.Before(from) {
		from, to = to, from
	}
	return Window{From: from, To: to}
}

func FilterTime[T any](items []T, window Window, at func(T) time.Time) []T {
	result := make([]T, 0)
	for _, item := range items {
		value := at(item)
		if !window.From.IsZero() && value.Before(window.From) {
			continue
		}
		if !window.To.IsZero() && value.After(window.To) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func StableSort[T any](items []T, less func(T, T) bool) []T {
	result := append([]T(nil), items...)
	sort.SliceStable(result, func(i, j int) bool { return less(result[i], result[j]) })
	return result
}

func Unique[T comparable](items []T) []T {
	seen := map[T]bool{}
	result := make([]T, 0, len(items))
	for _, item := range items {
		if seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func Chunk[T any](items []T, size int) [][]T {
	if size < 1 {
		size = 1
	}
	result := make([][]T, 0)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		result = append(result, append([]T(nil), items[start:end]...))
	}
	return result
}

func positive(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func parseTime(value string, fallback time.Time) time.Time {
	if value == "" {
		return fallback
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.UTC()
	}
	return fallback
}
