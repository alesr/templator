package templator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	// matches {{.FieldName}} and {{ .FieldName }} patterns
	fieldPattern = regexp.MustCompile(`{{\s*\.([a-zA-Z][a-zA-Z0-9._]*)\s*}}`)
	// matches {{if .FieldName}} patterns
	ifPattern = regexp.MustCompile(`{{\s*if\s+\.([a-zA-Z][a-zA-Z0-9._]*)\s*}}`)
)

type ValidationError struct {
	TemplateName string
	FieldPath    string
	Err          error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("template '%s' validation error: '%s' - '%s'", e.TemplateName, e.FieldPath, e.Err)
}

// validateTemplateFields analyzes template content and validates
// that all referenced fields exist in the data type
func validateTemplateFields[T any](name, content string, dataType T) error {
	typ := reflect.TypeOf(dataType)
	fields := extractTemplateFields(content)

	for _, field := range fields {
		if err := validateField(typ, field); err != nil {
			return &ValidationError{
				TemplateName: name,
				FieldPath:    field,
				Err:          err,
			}
		}
	}
	return nil
}

// extractTemplateFields returns a list of field paths used in the template
func extractTemplateFields(content string) []string {
	var fields []string
	fieldMatches := fieldPattern.FindAllStringSubmatch(content, -1)
	ifMatches := ifPattern.FindAllStringSubmatch(content, -1)

	// Add direct field references
	for _, match := range fieldMatches {
		if len(match) > 1 {
			fields = append(fields, match[1])
		}
	}

	// Add fields from if statements
	for _, match := range ifMatches {
		if len(match) > 1 {
			fields = append(fields, match[1])
		}
	}
	return uniqueFields(fields)
}

// uniqueFields removes duplicate field names
func uniqueFields(fields []string) []string {
	seen := make(map[string]struct{})
	unique := make([]string, 0, len(fields))

	for _, field := range fields {
		if _, ok := seen[field]; !ok {
			seen[field] = struct{}{}
			unique = append(unique, field)
		}
	}
	return unique
}

// validateField checks if a field path exists in the given type
func validateField(typ reflect.Type, fieldPath string) error {
	if typ == nil {
		return fmt.Errorf("nil type")
	}

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct type, got %v", typ.Kind())
	}

	parts := strings.Split(fieldPath, ".")
	current := typ

	for _, part := range parts {
		field, found := current.FieldByName(part)
		if !found {
			return fmt.Errorf("field '%s' not found in type %s", part, current.Name())
		}
		current = field.Type
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}
	}
	return nil
}
