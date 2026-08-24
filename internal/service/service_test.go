package service

import (
	"context"
	"testing"
	"time"

	"pulsegrid/internal/analytics"
	"pulsegrid/internal/domain"
	"pulsegrid/internal/store"
)

// newTestApp wires up an App backed by an in-memory repository and analytics
// store, returning the handles the test needs to assert on side effects.
func newTestApp(t *testing.T) (*App, *store.Memory, *analytics.Store) {
	t.Helper()
	repo := store.NewMemory()
	metrics := analytics.New()
	app := New(Config{Repository: repo, Metrics: metrics})
	return app, repo, metrics
}

// seedAutomation creates an enabled automation whose actions run a tag-contact
// action followed by a metric action. The tag-contact action is the one that
// historically panicked when the contact event carried no contact id.
func seedAutomation(t *testing.T, repo *store.Memory, trigger string) domain.Automation {
	t.Helper()
	now := time.Now().UTC()
	org := domain.NewOrganization("org-1", "Acme", "alice", now)
	if err := repo.CreateOrganization(context.Background(), org); err != nil {
		t.Fatalf("create organization: %v", err)
	}
	automation := domain.NewAutomation("auto-1", org.ID, "tagger", trigger, now)
	automation.Actions = []domain.Action{
		{Type: "tag-contact", Value: "lead"},
		{Type: "metric", Value: "automation.tagged"},
	}
	if err := repo.CreateAutomation(context.Background(), automation); err != nil {
		t.Fatalf("create automation: %v", err)
	}
	return automation
}

// A contact event missing the contact id must not crash automation processing.
// Before the fix, the tag-contact action called RequiredEventContactID, which
// panicked on the nil type assertion, killing the automation goroutine and
// skipping every later action (here: the metric record).
func TestProcessEvent_MissingContactID_DoesNotPanic(t *testing.T) {
	app, repo, metrics := newTestApp(t)
	seedAutomation(t, repo, "contact.tagged")

	// No ContactID and no Data["contact_id"] — the malformed input.
	event := domain.Event{
		ID:             "event-1",
		OrganizationID: "org-1",
		Type:           "contact.tagged",
		OccurredAt:     time.Now().UTC(),
	}

	// This used to panic; now it must complete cleanly.
	err := app.ProcessEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("ProcessEvent returned error for malformed event: %v", err)
	}

	// The follow-up metric action must still have run, proving the loop did not
	// abort after the failing tag-contact action.
	recorded := metrics.List(analytics.Filter{OrganizationID: "org-1", Name: "automation.tagged"})
	if len(recorded) != 1 {
		t.Fatalf("expected follow-up metric to be recorded, got %d metrics", len(recorded))
	}
}

// When the contact id is present on the event, the tag-contact action must
// still apply the tag, confirming the happy path is unaffected by the fix.
func TestProcessEvent_WithContactID_TagsContact(t *testing.T) {
	app, repo, metrics := newTestApp(t)
	seedAutomation(t, repo, "contact.tagged")

	now := time.Now().UTC()
	contact := domain.NewContact("contact-1", "org-1", "person@example.com", "Person", now)
	if err := repo.CreateContact(context.Background(), contact); err != nil {
		t.Fatalf("create contact: %v", err)
	}

	event := domain.Event{
		ID:             "event-1",
		OrganizationID: "org-1",
		Type:           "contact.tagged",
		ContactID:      "contact-1",
		OccurredAt:     now,
	}

	if err := app.ProcessEvent(context.Background(), event); err != nil {
		t.Fatalf("ProcessEvent returned error: %v", err)
	}

	updated, err := repo.GetContact(context.Background(), "contact-1")
	if err != nil {
		t.Fatalf("get contact: %v", err)
	}
	if !contains(updated.Tags, "lead") {
		t.Errorf("expected contact to be tagged \"lead\", got %v", updated.Tags)
	}

	recorded := metrics.List(analytics.Filter{OrganizationID: "org-1", Name: "automation.tagged"})
	if len(recorded) != 1 {
		t.Fatalf("expected follow-up metric to be recorded, got %d metrics", len(recorded))
	}
}

// A non-string contact_id entry in Data must also be handled gracefully
// rather than panicking the type assertion.
func TestProcessEvent_NonStringContactID_DoesNotPanic(t *testing.T) {
	app, repo, metrics := newTestApp(t)
	seedAutomation(t, repo, "contact.tagged")

	event := domain.Event{
		ID:             "event-1",
		OrganizationID: "org-1",
		Type:           "contact.tagged",
		OccurredAt:     time.Now().UTC(),
		Data:           map[string]any{"contact_id": 12345}, // wrong type, must not panic
	}

	if err := app.ProcessEvent(context.Background(), event); err != nil {
		t.Fatalf("ProcessEvent returned error for malformed event: %v", err)
	}

	recorded := metrics.List(analytics.Filter{OrganizationID: "org-1", Name: "automation.tagged"})
	if len(recorded) != 1 {
		t.Fatalf("expected follow-up metric to be recorded, got %d metrics", len(recorded))
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
