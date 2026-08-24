package httpapi

import (
	"errors"
	"net/http"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/jobs"
)

func StatusForError(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyExists), errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInvalidState):
		return http.StatusConflict
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, jobs.ErrUnavailable), errors.Is(err, jobs.ErrNotStarted):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// RetryableError reports whether err represents a transient backend failure
// that the caller may retry, and returns the recommended Retry-After hint in
// seconds when it is.
func RetryableError(err error) (bool, int) {
	if jobs.Retryable(err) {
		return true, retryAfterSeconds
	}
	return false, 0
}

// retryAfterSeconds is the Retry-After hint returned for transient outages. It
// is short because the in-memory queue recovers as soon as workers come back
// online; callers re-issuing the request idempotently will simply re-enqueue
// any remaining deliveries.
const retryAfterSeconds = 5

