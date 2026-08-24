package events

import (
	"strings"

	"pulsegrid/internal/domain"
)

func NormalizeType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func EventSubject(value domain.Event) string {
	if value.Subject != "" {
		return value.Subject
	}
	return value.ID
}
