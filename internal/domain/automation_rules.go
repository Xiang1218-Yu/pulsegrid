package domain

func ActionNames(value Automation) []string {
	result := make([]string, 0, len(value.Actions))
	for _, action := range value.Actions {
		result = append(result, action.Type)
	}
	return result
}

func AutomationIsActive(value Automation) bool {
	return value.Status == WorkflowEnabled
}
