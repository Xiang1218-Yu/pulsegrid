package service

import (
	"context"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
	"pulsegrid/internal/store"
)

// Source markers: publish(ctx, event) ProcessEvent Events.Publish Queue.Enqueue
func TestBug007ContextLifecycleCancelledEvent(t *testing.T) {
	app, organizationID, contactID := newBug007App(t)
	if _, err := app.CreateAutomation(context.Background(), AutomationInput{
		OrganizationID: organizationID,
		Name:           "unsubscribe on contact event",
		Trigger:        "contact.created",
		Actions:        []domain.Action{{Type: "unsubscribe"}},
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app.publish(ctx, events.NewEvent(organizationID, "contact.created", contactID, contactID, nil))

	time.Sleep(50 * time.Millisecond)
	contact, err := app.GetContact(context.Background(), contactID)
	if err != nil {
		t.Fatal(err)
	}
	if contact.Status != domain.ContactSubscribed {
		t.Fatalf("cancelled event changed contact status to %q", contact.Status)
	}
}

func TestBug007ContextLifecycleActiveEvent(t *testing.T) {
	app, organizationID, contactID := newBug007App(t)
	if _, err := app.CreateAutomation(context.Background(), AutomationInput{
		OrganizationID: organizationID,
		Name:           "unsubscribe on contact event",
		Trigger:        "contact.created",
		Actions:        []domain.Action{{Type: "unsubscribe"}},
	}); err != nil {
		t.Fatal(err)
	}

	app.publish(context.Background(), events.NewEvent(organizationID, "contact.created", contactID, contactID, nil))
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		contact, err := app.GetContact(context.Background(), contactID)
		if err != nil {
			t.Fatal(err)
		}
		if contact.Status == domain.ContactUnsubscribed {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("active event did not run automation")
}

func newBug007App(t *testing.T) (*App, string, string) {
	t.Helper()
	repository := store.NewMemory()
	app := New(Config{Repository: repository, Events: events.New(events.Config{Buffer: 4})})
	organization, err := app.CreateOrganization(context.Background(), OrganizationInput{Name: "Acme", Owner: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	contact := domain.NewContact("contact-1", organization.ID, "person@example.com", "Person", time.Now().UTC())
	if err := repository.CreateContact(context.Background(), contact); err != nil {
		t.Fatal(err)
	}
	return app, organization.ID, contact.ID
}
