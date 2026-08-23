package messaging

import (
	"strings"

	"pulsegrid/internal/domain"
)

func PreviewText(template domain.Template, values map[string]string) string {
	value := template.Subject + "\n" + template.Body
	for key, replacement := range values {
		value = strings.ReplaceAll(value, "{{"+key+"}}", replacement)
	}
	return value
}

func HasUnresolvedVariables(value string) bool {
	return strings.Contains(value, "{{") && strings.Contains(value, "}}")
}
