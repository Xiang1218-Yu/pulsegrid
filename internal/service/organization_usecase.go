package service

import (
	"context"

	"pulsegrid/internal/domain"
)

func OrganizationReady(ctx context.Context, value domain.Organization) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return domain.OrganizationIsOperational(value)
	}
}

func ContextAllowsWork(ctx context.Context) bool {
	return ctx != nil && ctx.Err() == nil
}

func OrganizationPlan(value domain.Organization) string {
	if value.Plan == "" {
		return "starter"
	}
	return value.Plan
}
