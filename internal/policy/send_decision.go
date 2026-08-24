package policy

import "pulsegrid/internal/domain"

func ExplainContactDecision(contact domain.Contact) string {
	switch contact.Status {
	case domain.ContactSubscribed:
		return "contact is subscribed"
	case domain.ContactUnsubscribed:
		return "contact unsubscribed"
	case domain.ContactBounced:
		return "contact bounced"
	case domain.ContactArchived:
		return "contact archived"
	default:
		return "unknown contact status"
	}
}

func IsSuppressed(decision Decision) bool {
	return !decision.Allowed
}
