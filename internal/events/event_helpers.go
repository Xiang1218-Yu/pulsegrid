package events

import (
	"context"
	"strings"

	"pulsegrid/internal/domain"
)

func ContextEnded(ctx context.Context) bool {
	return ctx != nil && ctx.Err() != nil
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
