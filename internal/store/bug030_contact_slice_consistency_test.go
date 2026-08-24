package store_test

import (
	"context"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

// Contact.Clone, ListContacts, SortContactsByEngagement
func TestBug030ContactSliceConsistencyKeepsState(t *testing.T) {
	repository := store.NewMemory()
	organization := domain.NewOrganization("org-1", "Acme", "owner", time.Now().UTC())
	if err := repository.CreateOrganization(context.Background(), organization); err != nil {
		t.Fatalf("create organization: %v", err)
	}
	contact := domain.NewContact("contact-1", organization.ID, "person@example.com", "Person", time.Now().UTC())
	contact.Tags = []string{"customer"}
	if err := repository.CreateContact(context.Background(), contact); err != nil {
		t.Fatalf("create contact: %v", err)
	}

	rows, err := repository.ListContacts(context.Background(), store.ContactFilter{OrganizationID: organization.ID})
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	sorted := service.SortContactsByEngagement(rows)
	sorted[0].Tags[0] = "mutated"

	again, err := repository.ListContacts(context.Background(), store.ContactFilter{OrganizationID: organization.ID})
	if err != nil {
		t.Fatalf("list contacts again: %v", err)
	}
	if again[0].Tags[0] != "customer" {
		t.Fatalf("stored contact tag = %q, want customer", again[0].Tags[0])
	}
}

func TestBug030ContactSliceConsistencyRegression(t *testing.T) {
	repository := store.NewMemory()
	organization := domain.NewOrganization("org-1", "Acme", "owner", time.Now().UTC())
	if err := repository.CreateOrganization(context.Background(), organization); err != nil {
		t.Fatalf("create organization: %v", err)
	}
	for _, name := range []string{"A", "B"} {
		contact := domain.NewContact("contact-"+name, organization.ID, name+"@example.com", name, time.Now().UTC())
		if err := repository.CreateContact(context.Background(), contact); err != nil {
			t.Fatalf("create contact %s: %v", name, err)
		}
	}
	rows, err := repository.ListContacts(context.Background(), store.ContactFilter{OrganizationID: organization.ID})
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("contact count = %d, want 2", len(rows))
	}
}
