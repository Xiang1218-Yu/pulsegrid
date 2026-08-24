package domain

import (
	"errors"
	"fmt"
)

// ErrMissingContactID is returned when an event carries no contact id, either
// on the dedicated ContactID field or under the legacy "contact_id" data key.
var ErrMissingContactID = errors.New("event missing contact_id")

// EventDataString reads a string field from event data. A missing key or a
// non-string value resolves to an empty string instead of panicking.
func EventDataString(value Event, key string) string {
	raw, ok := value.Data[key]
	if !ok {
		return ""
	}
	if str, ok := raw.(string); ok {
		return str
	}
	return fmt.Sprint(raw)
}

// RequiredEventContactID resolves the contact id associated with an event.
// The canonical location is Event.ContactID; the "contact_id" entry in
// Event.Data is honored as a legacy fallback. The function never panics on
// malformed input — a missing or non-string value yields ErrMissingContactID
// so callers can skip the action instead of crashing the automation.
func RequiredEventContactID(value Event) (string, error) {
	if value.ContactID != "" {
		return value.ContactID, nil
	}
	if raw, ok := value.Data["contact_id"]; ok {
		if str, ok := raw.(string); ok && str != "" {
			return str, nil
		}
	}
	return "", ErrMissingContactID
}

func EventName(value Event) string {
	if value.Type == "" {
		return "unknown"
	}
	return value.Type
}
