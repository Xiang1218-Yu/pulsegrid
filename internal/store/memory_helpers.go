package store

import (
	"strings"

	"pulsegrid/internal/domain"
)

func StoreContactMatches(value domain.Contact, search string) bool {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value.Email), search) ||
		strings.Contains(strings.ToLower(value.Name), search) ||
		strings.Contains(strings.ToLower(value.Company), search)
}

func StoreCampaignMatches(value domain.Campaign, search string) bool {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value.Name), search) ||
		strings.Contains(strings.ToLower(value.Description), search)
}
