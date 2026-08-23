package events

import (
	"context"
	"strings"

	"pulsegrid/internal/domain"
)

func AsyncContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func NormalizeType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func EventSubject(value domain.Event) string {
	if value.Subject != "" {
		return value.Subject
	}
	return value.ID
}
