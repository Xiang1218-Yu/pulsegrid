package pagination

import "strings"

func EncodeCursor(value string) string {
	return strings.TrimSpace(value)
}

func DecodeCursor(value string) string {
	return strings.TrimSpace(value)
}

func HasCursor(value string) bool {
	return DecodeCursor(value) != ""
}

// cloneWindow returns an independent copy of the paged window.
// Apply/ApplyCursor slice the source list with items[start:end], which only
// produces a new slice header that shares the underlying array with the source
// (and with every other page). Returning that slice directly lets mutations of
// the current page bleed back into the source list and adjacent pages — the
// order and contents of later pages get rewritten after a refresh. Copying into
// a fresh backing array keeps the paging boundaries (start/end) intact while
// guaranteeing the result is detached from the source.
func cloneWindow[T any](items []T) []T {
	if len(items) == 0 {
		return []T{}
	}
	out := make([]T, len(items))
	copy(out, items)
	return out
}
