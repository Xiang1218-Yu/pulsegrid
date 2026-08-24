package domain

import "testing"

func TestDeliveryEventShouldCount(t *testing.T) {
	cases := []struct {
		name   string
		before MessageStatus
		after  MessageStatus
		want   bool
	}{
		{"first send", MessageQueued, MessageSent, true},
		{"send to delivered", MessageSent, MessageDelivered, true},
		{"delivered to opened", MessageDelivered, MessageOpened, true},
		{"opened to clicked", MessageOpened, MessageClicked, true},

		// Duplicate identical callbacks must not count again — this is the
		// regression guard against providers resending the same webhook.
		{"duplicate sent", MessageSent, MessageSent, false},
		{"duplicate delivered", MessageDelivered, MessageDelivered, false},
		{"duplicate opened", MessageOpened, MessageOpened, false},
		{"duplicate clicked", MessageClicked, MessageClicked, false},

		// Backward / lateral transitions never count.
		{"delivered back to sent", MessageDelivered, MessageSent, false},
		{"opened back to delivered", MessageOpened, MessageDelivered, false},
		{"queued to failed", MessageQueued, MessageFailed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DeliveryEventShouldCount(
				Delivery{Status: tc.before},
				Delivery{Status: tc.after},
			)
			if got != tc.want {
				t.Fatalf("DeliveryEventShouldCount(%s->%s) = %v, want %v", tc.before, tc.after, got, tc.want)
			}
		})
	}
}
