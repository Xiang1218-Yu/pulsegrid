package service

import (
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/store"
)

func TestBug016AudienceSliceKeepsAllSubscribers(t *testing.T) {
	now := time.Now().UTC()
	contacts := []domain.Contact{
		domain.NewContact("contact-1", "org-1", "one@example.com", "One", now),
		domain.NewContact("contact-2", "org-1", "two@example.com", "Two", now.Add(time.Second)),
	}

	filtered := store.FilterSubscribed(contacts)
	if len(filtered) != 2 || filtered[0].ID != "contact-1" || filtered[1].ID != "contact-2" {
		t.Fatalf("audience contacts = %#v; want both subscribed contacts", filtered)
	}

	plan := BuildDeliveryPlan(domain.Campaign{ID: "campaign-1", TemplateID: "template-1"}, contacts)
	if plan.Count != 2 || len(plan.Contacts) != 2 {
		t.Fatalf("delivery plan count = %d with contacts %#v; want two contacts", plan.Count, plan.Contacts)
	}
}

func TestBug016EmptyAudienceStillProducesEmptyPlan(t *testing.T) {
	plan := BuildDeliveryPlan(domain.Campaign{ID: "campaign-1"}, nil)
	if !PlanIsEmpty(plan) || plan.Count != 0 {
		t.Fatalf("empty delivery plan = %#v; want no contacts", plan)
	}
}
