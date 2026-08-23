package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

// Coverage markers: POST /v1/contacts/{id}/unsubscribe, GET /v1/contacts, App.UnsubscribeContact, Contact.Unsubscribe.
func TestBug004UnsubscribeStateAcrossViews(t *testing.T) {
	repository := store.NewMemory()
	app := service.New(service.Config{Repository: repository})
	if err := repository.CreateOrganization(context.Background(), domain.NewOrganization("org-1", "Acme", "owner", time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	contact, err := app.CreateContact(context.Background(), service.ContactInput{
		OrganizationID: "org-1", Email: "person@example.com", Name: "Person",
	})
	if err != nil {
		t.Fatal(err)
	}
	server := New(Config{App: app})
	request := httptest.NewRequest(http.MethodPost, "/v1/contacts/"+contact.ID+"/unsubscribe", bytes.NewReader(nil))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unsubscribe status = %d, want %d", response.Code, http.StatusOK)
	}
	if bytes.Contains(response.Body.Bytes(), []byte(`"status":"subscribed"`)) {
		t.Fatalf("unsubscribe response still reports subscribed: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"unsubscribed"`)) {
		t.Fatalf("unsubscribe response does not report unsubscribed: %s", response.Body.String())
	}
	listRequest := httptest.NewRequest(http.MethodGet, "/v1/contacts?organization_id=org-1&status=subscribed", nil)
	listResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(listResponse, listRequest)
	if bytes.Contains(listResponse.Body.Bytes(), []byte(contact.ID)) {
		t.Fatalf("subscribed view still contains unsubscribed contact: %s", listResponse.Body.String())
	}
}

func TestBug004SubscribedContactRemainsVisible(t *testing.T) {
	repository := store.NewMemory()
	app := service.New(service.Config{Repository: repository})
	if err := repository.CreateOrganization(context.Background(), domain.NewOrganization("org-1", "Acme", "owner", time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateContact(context.Background(), service.ContactInput{
		OrganizationID: "org-1", Email: "active@example.com", Name: "Active",
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := app.ListContacts(context.Background(), store.ContactFilter{
		OrganizationID: "org-1", Status: string(domain.ContactSubscribed), Limit: 10,
	})
	if err != nil || len(rows) != 1 {
		t.Fatalf("subscribed contact list = %d, err=%v; want one contact", len(rows), err)
	}
}
