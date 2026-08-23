package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/store"
)

const bug15SourceMarkers = "RegisterDeliveryEvent delivery status"

func newBug15Fixture(t *testing.T, deliveryStatus domain.MessageStatus) (*App, *store.Memory, domain.Campaign, domain.Delivery) {
	t.Helper()
	repository := store.NewMemory()
	app := New(Config{Repository: repository, Logger: slog.Default()})
	now := time.Now().UTC()
	organization := domain.NewOrganization("org-15", "Acme", "alice", now)
	if err := repository.CreateOrganization(t.Context(), organization); err != nil {
		t.Fatalf("create organization: %v", err)
	}
	audience := domain.NewAudience("audience-15", organization.ID, "All contacts", now)
	if err := repository.CreateAudience(t.Context(), audience); err != nil {
		t.Fatalf("create audience: %v", err)
	}
	template := domain.NewTemplate("template-15", organization.ID, "Notice", now)
	template.Subject = "Notice"
	template.Body = "Hello"
	if err := repository.CreateTemplate(t.Context(), template); err != nil {
		t.Fatalf("create template: %v", err)
	}
	contact := domain.NewContact("contact-15", organization.ID, "contact15@example.com", "Contact 15", now)
	if err := repository.CreateContact(t.Context(), contact); err != nil {
		t.Fatalf("create contact: %v", err)
	}
	campaign := domain.NewCampaign("campaign-15", organization.ID, "Notice campaign", now)
	campaign.AudienceID = audience.ID
	campaign.TemplateID = template.ID
	campaign.SentCount = 1
	if err := repository.CreateCampaign(t.Context(), campaign); err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	delivery := domain.Delivery{
		ID: "delivery-15", CampaignID: campaign.ID, ContactID: "contact-15",
		TemplateID: template.ID, Channel: "email", Status: deliveryStatus,
		QueuedAt: now, UpdatedAt: now,
	}
	if err := repository.CreateDelivery(t.Context(), delivery); err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	return app, repository, campaign, delivery
}

func TestBug15StateConsistencyRejectsStaleDeliveryEvent(t *testing.T) {
	t.Log(bug15SourceMarkers)
	app, repository, campaign, delivery := newBug15Fixture(t, domain.MessageClicked)
	_, err := app.RegisterDeliveryEvent(context.Background(), delivery.ID, domain.MessageSent)
	if err == nil {
		t.Fatal("stale delivery event was accepted")
	}
	value, err := repository.GetCampaign(t.Context(), campaign.ID)
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if value.SentCount != campaign.SentCount {
		t.Fatalf("campaign sent count = %d, want %d", value.SentCount, campaign.SentCount)
	}
}

func TestBug15ForwardDeliveryEventUpdatesCampaign(t *testing.T) {
	app, repository, campaign, delivery := newBug15Fixture(t, domain.MessageQueued)
	_, err := app.RegisterDeliveryEvent(context.Background(), delivery.ID, domain.MessageSent)
	if err != nil {
		t.Fatalf("forward delivery event: %v", err)
	}
	value, err := repository.GetCampaign(t.Context(), campaign.ID)
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if value.SentCount != campaign.SentCount+1 {
		t.Fatalf("campaign sent count = %d, want %d", value.SentCount, campaign.SentCount+1)
	}
}
