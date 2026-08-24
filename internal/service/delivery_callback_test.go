package service

import (
	"context"
	"testing"
	"time"

	"pulsegrid/internal/analytics"
	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
	"pulsegrid/internal/store"
)

// newAppWithFixture wires a service against an in-memory store, event bus and
// analytics store, then seeds a running campaign with a single queued
// delivery. It returns the app plus the delivery id so a test can drive
// callbacks through RegisterDeliveryEvent.
func newAppWithFixture(t *testing.T) (*App, string, string) {
	t.Helper()
	ctx := context.Background()
	repo := store.NewMemory()
	bus := events.New(events.Config{Buffer: 32})
	metrics := analytics.New()
	app := New(Config{Repository: repo, Events: bus, Metrics: metrics})

	now := time.Now().UTC()
	org := domain.NewOrganization("org-1", "Acme", "alice", now)
	if err := repo.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("create org: %v", err)
	}
	tpl := domain.NewTemplate("tpl-1", org.ID, "Welcome", now)
	tpl.Subject = "Hi"
	tpl.Body = "Hello {{name}}"
	tpl.Published = true
	if err := repo.CreateTemplate(ctx, tpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	aud := domain.NewAudience("aud-1", org.ID, "All", now)
	if err := repo.CreateAudience(ctx, aud); err != nil {
		t.Fatalf("create audience: %v", err)
	}
	contact := domain.NewContact("contact-1", org.ID, "person@example.com", "Person", now)
	if err := repo.CreateContact(ctx, contact); err != nil {
		t.Fatalf("create contact: %v", err)
	}
	campaign := domain.NewCampaign("camp-1", org.ID, "Launch", now)
	campaign.AudienceID = aud.ID
	campaign.TemplateID = tpl.ID
	campaign.Status = domain.CampaignRunning
	campaign.TargetCount = 1
	if err := repo.CreateCampaign(ctx, campaign); err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	delivery := domain.Delivery{
		ID: "del-1", CampaignID: campaign.ID, ContactID: "contact-1",
		TemplateID: tpl.ID, Channel: "email",
		Status: domain.MessageQueued, QueuedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateDelivery(ctx, delivery); err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	return app, delivery.ID, org.ID
}

// TestRegisterDeliveryEventDuplicateDoesNotDoubleCount drives the same "sent"
// callback twice and asserts the campaign counter, analytics metric and
// published event each advance exactly once.
func TestRegisterDeliveryEventDuplicateDoesNotDoubleCount(t *testing.T) {
	app, deliveryID, orgID := newAppWithFixture(t)
	ctx := context.Background()

	sub := app.config.Events.Subscribe(events.TopicDeliverySent)
	defer sub.Close()

	// First "sent" callback: queued -> sent. Should count once.
	if _, err := app.RegisterDeliveryEvent(ctx, deliveryID, domain.MessageSent); err != nil {
		t.Fatalf("first sent: %v", err)
	}
	// Duplicate "sent" callback: sent -> sent. Must be a no-op for side effects.
	if _, err := app.RegisterDeliveryEvent(ctx, deliveryID, domain.MessageSent); err != nil {
		t.Fatalf("duplicate sent: %v", err)
	}

	campaign, err := app.config.Repository.GetCampaign(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaign.SentCount != 1 {
		t.Errorf("SentCount = %d, want 1 (duplicate callback inflated it)", campaign.SentCount)
	}

	summaries := app.config.Metrics.Summarize(analytics.Filter{OrganizationID: orgID, Name: "delivery.sent"})
	if len(summaries) != 1 || summaries[0].Sum != 1 {
		t.Errorf("delivery.sent metric sum = %v, want 1", summaries)
	}

	// Exactly one delivery.sent event should have been published.
	var got int
drain:
	for {
		select {
		case _, ok := <-sub.Events():
			if !ok {
				break drain
			}
			got++
		default:
			break drain
		}
	}
	if got != 1 {
		t.Errorf("published %d delivery.sent events, want 1", got)
	}
}

// TestRegisterDeliveryEventProgressStillCounts confirms that legitimate forward
// transitions still advance counters after the duplicate fix.
func TestRegisterDeliveryEventProgressStillCounts(t *testing.T) {
	app, deliveryID, _ := newAppWithFixture(t)
	ctx := context.Background()

	if _, err := app.RegisterDeliveryEvent(ctx, deliveryID, domain.MessageSent); err != nil {
		t.Fatalf("sent: %v", err)
	}
	if _, err := app.RegisterDeliveryEvent(ctx, deliveryID, domain.MessageDelivered); err != nil {
		t.Fatalf("delivered: %v", err)
	}

	campaign, err := app.config.Repository.GetCampaign(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaign.SentCount != 1 {
		t.Errorf("SentCount = %d, want 1", campaign.SentCount)
	}
	if campaign.DeliveredCount != 1 {
		t.Errorf("DeliveredCount = %d, want 1", campaign.DeliveredCount)
	}
}

// TestRegisterDeliveryEventDuplicateDelivered covers the delivered status
// specifically (the path the report describes), asserting the delivered
// counter cannot be inflated by a resent webhook.
func TestRegisterDeliveryEventDuplicateDelivered(t *testing.T) {
	app, deliveryID, _ := newAppWithFixture(t)
	ctx := context.Background()

	if _, err := app.RegisterDeliveryEvent(ctx, deliveryID, domain.MessageSent); err != nil {
		t.Fatalf("sent: %v", err)
	}
	if _, err := app.RegisterDeliveryEvent(ctx, deliveryID, domain.MessageDelivered); err != nil {
		t.Fatalf("delivered: %v", err)
	}
	// Provider resends the same "delivered" webhook twice.
	if _, err := app.RegisterDeliveryEvent(ctx, deliveryID, domain.MessageDelivered); err != nil {
		t.Fatalf("duplicate delivered: %v", err)
	}
	if _, err := app.RegisterDeliveryEvent(ctx, deliveryID, domain.MessageDelivered); err != nil {
		t.Fatalf("duplicate delivered 2: %v", err)
	}

	campaign, err := app.config.Repository.GetCampaign(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaign.DeliveredCount != 1 {
		t.Errorf("DeliveredCount = %d, want 1 (duplicate delivered callback inflated it)", campaign.DeliveredCount)
	}
	if campaign.SentCount != 1 {
		t.Errorf("SentCount = %d, want 1", campaign.SentCount)
	}
}
