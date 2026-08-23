package ratelimit

import "strings"

func OrganizationKey(organizationID string) string {
	return "organization:" + strings.TrimSpace(organizationID)
}

func ContactKey(contactID string) string {
	return "contact:" + strings.TrimSpace(contactID)
}

func EndpointKey(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + ":" + strings.TrimSpace(path)
}
