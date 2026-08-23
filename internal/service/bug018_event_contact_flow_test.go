package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
	"pulsegrid/internal/store"
)

func TestBug018EventContactFlowReachesAutomation(t *testing.T) {
	app, repository, organization, contact := newBug018App(t)
	event := events.NewEvent(
		organization.ID,
		"contact.created",
		contact.ID,
		contact.ID,
		map[string]any{"contact_id": contact.ID},
	)
	if err := app.ProcessEvent(context.Background(), event); err != nil {
		t.Fatalf("ProcessEvent returned error: %v", err)
	}
	updated, err := repository.GetContact(context.Background(), contact.ID)
	if err != nil {
		t.Fatalf("Repository.GetContact: %v", err)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "vip" {
		t.Fatalf("contact tags after events.NewEvent -> ProcessEvent = %#v; want vip", updated.Tags)
	}
}

func TestBug018DirectAutomationEventStillWorks(t *testing.T) {
	app, repository, organization, contact := newBug018App(t)
	event := domain.Event{
		OrganizationID: organization.ID,
		Type:           "contact.created",
		Subject:        contact.ID,
		ContactID:      contact.ID,
		Data:           map[string]any{"contact_id": contact.ID},
		OccurredAt:     time.Now().UTC(),
	}
	if err := app.ProcessEvent(context.Background(), event); err != nil {
		t.Fatalf("ProcessEvent returned error: %v", err)
	}
	updated, err := repository.GetContact(context.Background(), contact.ID)
	if err != nil {
		t.Fatalf("Repository.GetContact: %v", err)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "vip" {
		t.Fatalf("direct ProcessEvent contact tags = %#v; want vip", updated.Tags)
	}
}

func newBug018App(t *testing.T) (*App, *store.Memory, domain.Organization, domain.Contact) {
	t.Helper()
	repository := store.NewMemory()
	app := New(Config{Repository: repository, Logger: slog.Default()})
	now := time.Now().UTC()
	organization := domain.NewOrganization("org-018", "Flow Org", "owner", now)
	if err := repository.CreateOrganization(context.Background(), organization); err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}
	contact := domain.NewContact("contact-018", organization.ID, "flow@example.com", "Flow", now)
	if err := repository.CreateContact(context.Background(), contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	automation := domain.NewAutomation("automation-018", organization.ID, "Tag new contacts", "contact.created", now)
	automation.Actions = []domain.Action{{Type: "tag-contact", Value: "vip"}}
	if err := repository.CreateAutomation(context.Background(), automation); err != nil {
		t.Fatalf("CreateAutomation: %v", err)
	}
	return app, repository, organization, contact
}
