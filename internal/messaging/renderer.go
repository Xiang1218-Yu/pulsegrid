package messaging

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"pulsegrid/internal/domain"
)

type Renderer struct {
	Functions map[string]func(string) string
}

type Preview struct {
	Subject   string   `json:"subject"`
	Body      string   `json:"body"`
	Channel   string   `json:"channel"`
	Variables []string `json:"variables"`
	Missing   []string `json:"missing,omitempty"`
}

type Batch struct {
	Messages []Preview `json:"messages"`
	Skipped  int       `json:"skipped"`
	Errors   []string  `json:"errors,omitempty"`
}

var variablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)(?:\|([a-zA-Z0-9_-]+))?\s*\}\}`)

func NewRenderer() Renderer {
	return Renderer{Functions: map[string]func(string) string{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
		"title": strings.Title,
	}}
}

func (r Renderer) Render(template domain.Template, values map[string]string) (Preview, error) {
	if err := template.Validate(); err != nil {
		return Preview{}, err
	}
	used := extractVariables(template.Subject + "\n" + template.Body)
	missing := make([]string, 0)
	for _, variable := range used {
		if _, ok := values[variable]; !ok {
			missing = append(missing, variable)
		}
	}
	if len(missing) > 0 {
		return Preview{Variables: used, Missing: missing}, fmt.Errorf("missing variables: %s", strings.Join(missing, ", "))
	}
	subject, err := r.renderText(template.Subject, values)
	if err != nil {
		return Preview{}, err
	}
	body, err := r.renderText(template.Body, values)
	if err != nil {
		return Preview{}, err
	}
	return Preview{Subject: subject, Body: body, Channel: template.Channel, Variables: used}, nil
}

func (r Renderer) RenderBatch(template domain.Template, contacts []domain.Contact, common map[string]string) Batch {
	result := Batch{Messages: make([]Preview, 0, len(contacts))}
	for _, contact := range contacts {
		values := copyStrings(common)
		values["email"] = contact.Email
		values["name"] = contact.Name
		values["company"] = contact.Company
		for key, value := range contact.Attributes {
			values[key] = value
		}
		preview, err := r.Render(template, values)
		if err != nil {
			result.Errors = append(result.Errors, contact.Email+": "+err.Error())
			result.Skipped++
			continue
		}
		result.Messages = append(result.Messages, preview)
	}
	return result
}

func (r Renderer) Validate(template domain.Template) []string {
	variables := extractVariables(template.Subject + "\n" + template.Body)
	return variables
}

func (r Renderer) renderText(source string, values map[string]string) (string, error) {
	return variablePattern.ReplaceAllStringFunc(source, func(match string) string {
		parts := variablePattern.FindStringSubmatch(match)
		value := values[parts[1]]
		if parts[2] != "" {
			if fn := r.Functions[parts[2]]; fn != nil {
				value = fn(value)
			}
		}
		return value
	}), nil
}

func extractVariables(source string) []string {
	seen := map[string]bool{}
	for _, parts := range variablePattern.FindAllStringSubmatch(source, -1) {
		if len(parts) > 1 {
			seen[parts[1]] = true
		}
	}
	result := make([]string, 0, len(seen))
	for variable := range seen {
		result = append(result, variable)
	}
	sort.Strings(result)
	return result
}

func CopyValues(source map[string]string) map[string]string { return copyStrings(source) }

func copyStrings(source map[string]string) map[string]string {
	if source == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for key, value := range source {
		out[key] = value
	}
	return out
}
