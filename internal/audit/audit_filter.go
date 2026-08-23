package audit

import "time"

func RecentFilter(limit int) Filter {
	if limit < 1 {
		limit = 100
	}
	return Filter{From: time.Now().UTC().Add(-24 * time.Hour), Limit: limit}
}

func OrganizationFilter(organizationID string, limit int) Filter {
	if limit < 1 {
		limit = 100
	}
	return Filter{OrganizationID: organizationID, Limit: limit}
}
