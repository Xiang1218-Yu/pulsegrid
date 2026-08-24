package domain

import "fmt"

func AudienceFilterCount(value Audience) int {
	return len(value.AllOf) + len(value.AnyOf) + len(value.Excluded)
}

func FilterSummary(value Filter) string {
	return fmt.Sprintf("%s %s %s", value.Field, value.Operator, value.Value)
}
