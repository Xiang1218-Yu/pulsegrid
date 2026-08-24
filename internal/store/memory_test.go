package store

import (
	"context"
	"sort"
	"testing"
	"time"

	"pulsegrid/internal/domain"
)

func newTestMemoryWithContacts(t *testing.T, now time.Time, orgID string, contacts ...domain.Contact) *Memory {
	t.Helper()
	mem := NewMemory()
	org := domain.NewOrganization("org-1", "Acme", "alice", now)
	if err := mem.CreateOrganization(context.Background(), org); err != nil {
		t.Fatalf("create organization: %v", err)
	}
	for i, c := range contacts {
		if c.ID == "" {
			contacts[i].ID = contactID(i)
		}
		contacts[i].OrganizationID = org.ID
		if err := mem.CreateContact(context.Background(), contacts[i].Clone()); err != nil {
			t.Fatalf("create contact %d: %v", i, err)
		}
	}
	return mem
}

func contactID(i int) string {
	switch i {
	case 0:
		return "contact-a"
	case 1:
		return "contact-b"
	default:
		return "contact-c"
	}
}

func TestListContactsReturnsIndependentCopy(t *testing.T) {
	now := time.Now().UTC()
	base := domain.Contact{
		ID:             "contact-a",
		OrganizationID: "org-1",
		Email:          "a@example.com",
		Name:           "Alice",
		Status:         domain.ContactSubscribed,
		Tags:           []string{"vip", "early"},
		Attributes:     map[string]string{"plan": "pro"},
		UpdatedAt:      now,
	}
	mem := newTestMemoryWithContacts(t, now, "org-1", base)

	rows, err := mem.ListContacts(context.Background(), ContactFilter{OrganizationID: "org-1", Limit: 100})
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(rows))
	}

	// Mutate the returned element in place: tags, attributes, and slice order.
	rows[0].Tags[0] = "mutated"
	rows[0].Attributes["plan"] = "tampered"
	rows[0].Status = domain.ContactArchived

	// A fresh query must not reflect any of those mutations.
	fresh, err := mem.ListContacts(context.Background(), ContactFilter{OrganizationID: "org-1", Limit: 100})
	if err != nil {
		t.Fatalf("ListContacts (re-read): %v", err)
	}
	if len(fresh) != 1 {
		t.Fatalf("expected 1 contact on re-read, got %d", len(fresh))
	}
	if got := fresh[0].Status; got != domain.ContactSubscribed {
		t.Fatalf("status leaked from caller mutation: got %q want %q", got, domain.ContactSubscribed)
	}
	if got, want := fresh[0].Tags, []string{"vip", "early"}; !equalTags(got, want) {
		t.Fatalf("tags leaked from caller mutation: got %v want %v", got, want)
	}
	if got := fresh[0].Attributes["plan"]; got != "pro" {
		t.Fatalf("attribute leaked from caller mutation: got %q want %q", got, "pro")
	}
}

func TestListContactsSortingDoesNotLeak(t *testing.T) {
	// Contacts ordered so the store's UpdatedAt-descending order is the reverse
	// of the order an engagement-style sort would produce. Mutating the returned
	// slice must not change what subsequent queries return.
	now := time.Now().UTC()
	earlier := now.Add(-2 * time.Hour)
	middle := now.Add(-1 * time.Hour)
	contacts := []domain.Contact{
		{ID: "contact-a", Email: "a@example.com", Name: "Alice", Status: domain.ContactSubscribed, LastEngagedAt: &earlier, Tags: []string{}, UpdatedAt: earlier},
		{ID: "contact-b", Email: "b@example.com", Name: "Bob", Status: domain.ContactSubscribed, LastEngagedAt: &middle, Tags: []string{}, UpdatedAt: middle},
		{ID: "contact-c", Email: "c@example.com", Name: "Carol", Status: domain.ContactSubscribed, LastEngagedAt: &now, Tags: []string{}, UpdatedAt: now},
	}
	mem := newTestMemoryWithContacts(t, now, "org-1", contacts...)

	rows, err := mem.ListContacts(context.Background(), ContactFilter{OrganizationID: "org-1", Limit: 100})
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 contacts, got %d", len(rows))
	}

	// Simulate a caller applying its own ordering on the returned slice.
	sort.Slice(rows, func(i, j int) bool { return rows[i].Email > rows[j].Email })

	// Re-query: the store's own ordering (UpdatedAt desc) must be intact.
	fresh, err := mem.ListContacts(context.Background(), ContactFilter{OrganizationID: "org-1", Limit: 100})
	if err != nil {
		t.Fatalf("ListContacts (re-read): %v", err)
	}
	wantOrder := []string{"contact-c", "contact-b", "contact-a"} // UpdatedAt desc
	for i, want := range wantOrder {
		if fresh[i].ID != want {
			t.Fatalf("order leaked at index %d: got %q want %q", i, fresh[i].ID, want)
		}
	}
}

func TestListContactsReturnsCompleteResultSet(t *testing.T) {
	now := time.Now().UTC()
	contacts := make([]domain.Contact, 0, 5)
	for i := 0; i < 5; i++ {
		c := domain.Contact{
			ID:     "contact-" + string(rune('a'+i)),
			Email:  string(rune('a'+i)) + "@example.com",
			Name:   string(rune('A'+i)) + "lice",
			Status: domain.ContactSubscribed,
			Tags:   []string{"bulk"},
			UpdatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		contacts = append(contacts, c)
	}
	mem := newTestMemoryWithContacts(t, now, "org-1", contacts...)

	rows, err := mem.ListContacts(context.Background(), ContactFilter{OrganizationID: "org-1", Limit: 100})
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("expected 5 contacts, got %d", len(rows))
	}
	seen := map[string]bool{}
	for _, c := range rows {
		seen[c.Email] = true
	}
	if len(seen) != 5 {
		t.Fatalf("duplicate or missing contacts in result: %v", seen)
	}
}

func TestGetContactReturnsIndependentCopy(t *testing.T) {
	now := time.Now().UTC()
	base := domain.Contact{
		ID:             "contact-a",
		Email:          "a@example.com",
		Name:           "Alice",
		Status:         domain.ContactSubscribed,
		Tags:           []string{"vip"},
		Attributes:     map[string]string{"plan": "pro"},
		UpdatedAt:      now,
	}
	mem := newTestMemoryWithContacts(t, now, "org-1", base)

	got, err := mem.GetContact(context.Background(), "contact-a")
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	got.Tags[0] = "mutated"
	got.Attributes["plan"] = "tampered"

	again, err := mem.GetContact(context.Background(), "contact-a")
	if err != nil {
		t.Fatalf("GetContact (re-read): %v", err)
	}
	if got, want := again.Tags, []string{"vip"}; !equalTags(got, want) {
		t.Fatalf("tags leaked from GetContact mutation: got %v want %v", got, want)
	}
	if again.Attributes["plan"] != "pro" {
		t.Fatalf("attribute leaked from GetContact mutation: got %q want %q", again.Attributes["plan"], "pro")
	}
}

func equalTags(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
