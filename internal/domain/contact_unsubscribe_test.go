package domain

import (
	"testing"
	"time"
)

func TestContactUnsubscribeSetsUnsubscribedState(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	contact := NewContact("c1", "org1", "person@example.com", "Person", now)

	if err := contact.Unsubscribe(now); err != nil {
		t.Fatalf("unsubscribe returned error: %v", err)
	}

	if contact.Status != ContactUnsubscribed {
		t.Fatalf("status = %q, want %q", contact.Status, ContactUnsubscribed)
	}
	if contact.SubscribedAt != nil {
		t.Fatalf("subscribed_at = %v, want nil", contact.SubscribedAt)
	}
	if contact.UnsubscribedAt == nil || !contact.UnsubscribedAt.Equal(now) {
		t.Fatalf("unsubscribed_at = %v, want %v", contact.UnsubscribedAt, now)
	}
	if !contact.UpdatedAt.Equal(now) {
		t.Fatalf("updated_at = %v, want %v", contact.UpdatedAt, now)
	}
}

// TestContactUnsubscribeExcludedFromSubscribedView guards the original bug: after
// unsubscribing, a contact must no longer match the subscribed-contact view that
// the list page and send preparation rely on, while a still-subscribed contact
// continues to match.
func TestContactUnsubscribeExcludedFromSubscribedView(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	subscribed := NewContact("c1", "org1", "subscribed@example.com", "Sub", now)
	unsubscribed := NewContact("c2", "org1", "unsubscribed@example.com", "Unsub", now)

	if err := unsubscribed.Unsubscribe(now); err != nil {
		t.Fatalf("unsubscribe returned error: %v", err)
	}

	if !ContactCanReceive(subscribed) {
		t.Fatalf("subscribed contact should be receivable")
	}
	if ContactCanReceive(unsubscribed) {
		t.Fatalf("unsubscribed contact should not be receivable")
	}
	if unsubscribed.Status != ContactUnsubscribed {
		t.Fatalf("status = %q, want %q", unsubscribed.Status, ContactUnsubscribed)
	}
	if subscribed.Status != ContactSubscribed {
		t.Fatalf("subscribed status changed unexpectedly: %q", subscribed.Status)
	}
}

func TestContactUnsubscribeIdempotentGuard(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	contact := NewContact("c1", "org1", "person@example.com", "Person", now)

	if err := contact.Unsubscribe(now); err != nil {
		t.Fatalf("first unsubscribe returned error: %v", err)
	}
	if err := contact.Unsubscribe(now.Add(time.Hour)); err != ErrInvalidState {
		t.Fatalf("second unsubscribe error = %v, want %v", err, ErrInvalidState)
	}
	if !contact.UnsubscribedAt.Equal(now) {
		t.Fatalf("unsubscribed_at changed on second unsubscribe: got %v, want %v", contact.UnsubscribedAt, now)
	}
}
