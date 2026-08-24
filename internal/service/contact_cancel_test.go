package service

import (
	"context"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
	"pulsegrid/internal/store"
)

func setupApp(t *testing.T) (*App, string, string) {
	t.Helper()
	repo := store.NewMemory()
	bus := events.New(events.Config{Buffer: 16, DropWhenBusy: true})
	app := New(Config{Repository: repo, Events: bus})
	ctx := context.Background()

	org := domain.NewOrganization("org-1", "Acme", "alice", time.Now().UTC())
	if err := repo.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("create organization: %v", err)
	}

	// Automation that rewrites contact state when contact.created fires.
	automation := domain.NewAutomation("auto-1", org.ID, "auto-unsubscribe", "contact.created", time.Now().UTC())
	automation.Actions = []domain.Action{{Type: "unsubscribe"}}
	if err := repo.CreateAutomation(ctx, automation); err != nil {
		t.Fatalf("create automation: %v", err)
	}
	return app, org.ID, automation.ID
}

func TestCreateContact_RunsAutomationOnNormalRequest(t *testing.T) {
	app, orgID, _ := setupApp(t)
	repo := app.config.Repository

	contact, err := app.CreateContact(context.Background(), ContactInput{
		OrganizationID: orgID, Email: "person@example.com", Name: "Person",
	})
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}

	// The normal request completes the full flow: the automation runs and the
	// contact's status is rewritten to unsubscribed.
	stored, err := repo.GetContact(context.Background(), contact.ID)
	if err != nil {
		t.Fatalf("get contact: %v", err)
	}
	if stored.Status != domain.ContactUnsubscribed {
		t.Fatalf("expected automation to unsubscribe contact on normal request, got %q", stored.Status)
	}
}

func TestProcessEvent_AbortsOnCancelledContext(t *testing.T) {
	app, orgID, _ := setupApp(t)
	repo := app.config.Repository

	// A contact already committed by an earlier (pre-cancellation) step.
	contact := domain.NewContact("contact-1", orgID, "person@example.com", "Person", time.Now().UTC())
	if err := repo.CreateContact(context.Background(), contact); err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel() // simulate a request the client abandoned before automation runs

	event := domain.Event{
		ID: "event-1", OrganizationID: orgID, Type: "contact.created",
		Subject: "contact.created", ContactID: contact.ID, OccurredAt: time.Now().UTC(),
	}
	if err := app.ProcessEvent(cancelled, event); err == nil {
		t.Fatalf("expected ProcessEvent to abort on cancelled context")
	}

	stored, err := repo.GetContact(context.Background(), contact.ID)
	if err != nil {
		t.Fatalf("get contact: %v", err)
	}
	if stored.Status != domain.ContactSubscribed {
		t.Fatalf("expected automation to leave contact state untouched on cancelled request, got %q", stored.Status)
	}
}
