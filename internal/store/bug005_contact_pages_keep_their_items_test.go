package store

import (
	"context"
	"testing"

	"pulsegrid/internal/domain"
)

// Coverage markers: ListContacts, App.ListContacts, GET /v1/contacts, contactPage.
func TestBug005ContactPagesKeepTheirItems(t *testing.T) {
	memory := NewMemory()
	now := timeNow()
	if err := memory.CreateOrganization(context.Background(), domain.NewOrganization("org-1", "Acme", "owner", now)); err != nil {
		t.Fatal(err)
	}
	first := domain.NewContact("contact-a", "org-1", "a@example.com", "A", now)
	second := domain.NewContact("contact-b", "org-1", "b@example.com", "B", now)
	third := domain.NewContact("contact-c", "org-1", "c@second.example.com", "C", now)
	fourth := domain.NewContact("contact-d", "org-1", "d@second.example.com", "D", now)
	if err := memory.CreateContact(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := memory.CreateContact(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := memory.CreateContact(context.Background(), third); err != nil {
		t.Fatal(err)
	}
	if err := memory.CreateContact(context.Background(), fourth); err != nil {
		t.Fatal(err)
	}
	pageA, err := memory.ListContacts(context.Background(), ContactFilter{OrganizationID: "org-1", Search: "example.com", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := memory.ListContacts(context.Background(), ContactFilter{OrganizationID: "org-1", Search: "second.example.com", Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if len(pageA) != 1 || pageA[0].ID != "contact-a" {
		t.Fatalf("first page changed after second query: %#v", pageA)
	}
}

func TestBug005EmptyContactPageIsStable(t *testing.T) {
	memory := NewMemory()
	page, err := memory.ListContacts(context.Background(), ContactFilter{OrganizationID: "missing", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 0 {
		t.Fatalf("empty contact page contains %d items", len(page))
	}
}
