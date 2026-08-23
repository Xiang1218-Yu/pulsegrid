package service

// Coverage markers: RegisterDeliveryEvent, SentCount, MessageSent, GetCampaign.

import (
	"context"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/store"
)

func TestBug020RepeatedDeliveryEventKeepsCampaignCount(t *testing.T) {
	app, repository, campaign, delivery := newBug020App(t)
	if _, err := app.RegisterDeliveryEvent(context.Background(), delivery.ID, domain.MessageSent); err != nil {
		t.Fatalf("first RegisterDeliveryEvent: %v", err)
	}
	if _, err := app.RegisterDeliveryEvent(context.Background(), delivery.ID, domain.MessageSent); err != nil {
		t.Fatalf("repeated RegisterDeliveryEvent: %v", err)
	}
	updated, err := repository.GetCampaign(context.Background(), campaign.ID)
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}
	if updated.SentCount != 1 {
		t.Fatalf("campaign sent count after repeated callback = %d; want 1", updated.SentCount)
	}
}

func TestBug020ForwardDeliveryEventAdvancesOnce(t *testing.T) {
	app, repository, campaign, delivery := newBug020App(t)
	if _, err := app.RegisterDeliveryEvent(context.Background(), delivery.ID, domain.MessageSent); err != nil {
		t.Fatalf("RegisterDeliveryEvent: %v", err)
	}
	updated, err := repository.GetCampaign(context.Background(), campaign.ID)
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}
	if updated.SentCount != 1 {
		t.Fatalf("campaign sent count after forward callback = %d; want 1", updated.SentCount)
	}
}

func newBug020App(t *testing.T) (*App, *store.Memory, domain.Campaign, domain.Delivery) {
	t.Helper()
	repository := store.NewMemory()
	app := New(Config{Repository: repository})
	now := time.Now().UTC()
	organization := domain.NewOrganization("org-020", "Consistency Org", "owner", now)
	if err := repository.CreateOrganization(context.Background(), organization); err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}
	audience := domain.NewAudience("audience-020", organization.ID, "All contacts", now)
	if err := repository.CreateAudience(context.Background(), audience); err != nil {
		t.Fatalf("CreateAudience: %v", err)
	}
	template := domain.NewTemplate("template-020", organization.ID, "Lifecycle template", now)
	template.Subject = "Lifecycle"
	template.Body = "Hello"
	if err := repository.CreateTemplate(context.Background(), template); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	campaign := domain.NewCampaign("campaign-020", organization.ID, "Lifecycle", now)
	campaign.AudienceID = "audience-020"
	campaign.TemplateID = "template-020"
	if err := repository.CreateCampaign(context.Background(), campaign); err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}
	contact := domain.NewContact("contact-020", organization.ID, "consistency@example.com", "Consistency", now)
	if err := repository.CreateContact(context.Background(), contact); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	delivery := domain.Delivery{
		ID: "delivery-020", CampaignID: campaign.ID, ContactID: "contact-020",
		TemplateID: campaign.TemplateID, Channel: "email", Status: domain.MessageQueued,
		QueuedAt: now, UpdatedAt: now,
	}
	if err := repository.CreateDelivery(context.Background(), delivery); err != nil {
		t.Fatalf("CreateDelivery: %v", err)
	}
	return app, repository, campaign, delivery
}
