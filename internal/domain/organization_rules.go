package domain

func OrganizationIsOperational(value Organization) bool {
	return value.Status == OrganizationActive
}

func OrganizationLabel(value Organization) string {
	if value.Name != "" {
		return value.Name
	}
	return value.ID
}
