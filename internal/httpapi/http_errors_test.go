package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/jobs"
)

func TestStatusForErrorMapsTransientsToServiceUnavailable(t *testing.T) {
	queuedErr := fmt.Errorf("queue delivery d-1: %w", jobs.ErrUnavailable)
	cases := map[error]int{
		jobs.ErrUnavailable:               http.StatusServiceUnavailable,
		queuedErr:                          http.StatusServiceUnavailable,
		jobs.ErrNotStarted:                 http.StatusServiceUnavailable,
		domain.ErrNotFound:                 http.StatusNotFound,
		domain.ErrInvalidInput:             http.StatusBadRequest,
		domain.ErrConflict:                 http.StatusConflict,
		domain.ErrInvalidState:             http.StatusConflict,
		errors.New("boom"):                http.StatusInternalServerError,
	}
	for err, want := range cases {
		if got := StatusForError(err); got != want {
			t.Errorf("StatusForError(%v) = %d, want %d", err, got, want)
		}
	}
}

func TestRetryableErrorSignalsRetryAfter(t *testing.T) {
	queuedErr := fmt.Errorf("queue delivery d-1: %w", jobs.ErrUnavailable)
	retryable, after := RetryableError(queuedErr)
	if !retryable {
		t.Fatalf("want retryable for ErrUnavailable")
	}
	if after <= 0 {
		t.Fatalf("retry-after = %d, want > 0", after)
	}
	retryable, _ = RetryableError(errors.New("unrelated"))
	if retryable {
		t.Fatalf("unrelated error must not be retryable")
	}
}

func TestWriteErrorEmitsRetryableBodyForUnavailable(t *testing.T) {
	queuedErr := fmt.Errorf("queue delivery d-1: %w", jobs.ErrUnavailable)
	rec := httptest.NewRecorder()
	writeError(rec, queuedErr)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Header().Get("retry-after"); got == "" {
		t.Fatalf("retry-after header missing")
	}
	var body struct {
		Error      string `json:"error"`
		Retryable  bool   `json:"retryable"`
		RetryAfter int    `json:"retry_after"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !body.Retryable {
		t.Fatalf("body.retryable = false, want true")
	}
	if body.RetryAfter <= 0 {
		t.Fatalf("body.retry_after = %d, want > 0", body.RetryAfter)
	}
}

// A genuine conflict (campaign already running) must stay 409 with no retry hint
// so callers do not treat state-machine rejections as transient outages.
func TestWriteErrorConflictIsNotRetryable(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, domain.ErrInvalidState)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if got := rec.Header().Get("retry-after"); got != "" {
		t.Fatalf("retry-after header = %q, want empty", got)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if _, ok := body["retryable"]; ok {
		t.Fatalf("conflict body must not carry retryable field, got %v", body)
	}
}
