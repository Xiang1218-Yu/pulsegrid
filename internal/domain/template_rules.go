package domain

import "strings"

func TemplateHasVariables(value Template) bool {
	return strings.Contains(value.Subject, "{{") || strings.Contains(value.Body, "{{")
}

func TemplateVariableSet(value Template) map[string]bool {
	result := map[string]bool{}
	for _, variable := range value.Variables {
		result[variable] = true
	}
	return result
}
