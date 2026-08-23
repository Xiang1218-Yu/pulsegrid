package service

import (
	"errors"
	"strings"

	"pulsegrid/internal/domain"
)

func ValidateOrganizationInput(input OrganizationInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.Join(domain.ErrInvalidInput, errors.New("organization name is required"))
	}
	if strings.TrimSpace(input.Owner) == "" {
		return errors.Join(domain.ErrInvalidInput, errors.New("organization owner is required"))
	}
	return nil
}

func ValidateContactInput(input ContactInput) error {
	if strings.TrimSpace(input.OrganizationID) == "" || !strings.Contains(input.Email, "@") {
		return errors.Join(domain.ErrInvalidInput, errors.New("organization and valid email are required"))
	}
	return nil
}

func ValidateCampaignInput(input CampaignInput) error {
	if input.OrganizationID == "" || input.AudienceID == "" || input.TemplateID == "" {
		return errors.Join(domain.ErrInvalidInput, errors.New("campaign references are required"))
	}
	return nil
}
