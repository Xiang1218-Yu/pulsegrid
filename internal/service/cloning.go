package service

import "pulsegrid/internal/domain"

func CloneOrganization(value domain.Organization) domain.Organization {
	return value.Clone()
}

func CloneContact(value domain.Contact) domain.Contact {
	return value.Clone()
}

func CloneTemplate(value domain.Template) domain.Template {
	return value.Clone()
}

func CloneCampaign(value domain.Campaign) domain.Campaign {
	return value.Clone()
}

func CloneAutomation(value domain.Automation) domain.Automation {
	return value.Clone()
}
