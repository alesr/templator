package templator

import (
	"fmt"
	"reflect"
	"text/template/parse"
)

type ValidationError struct {
	TemplateName string
	FieldPath    string
	Err          error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("template '%s' validation error: %s - %v", e.TemplateName, e.FieldPath, e.Err)
}

// validateTemplateFields analyzes a template and validates that all referenced fields exist in the data type
func validateTemplateFields[T any](name string, tmpl *parse.Tree, dataType T) error {
	typ := reflect.TypeOf(dataType)
	fields := extractTemplateFields(tmpl.Root)

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
func extractTemplateFields(node parse.Node) []string {
	var fields []string
	var visit func(node parse.Node)

	visit = func(node parse.Node) {
		if node == nil {
			return
		}

		switch n := node.(type) {
		case *parse.FieldNode:
			fields = append(fields, n.Ident[0])
		case *parse.ListNode: // Changed from parse.ListNode to *parse.ListNode
			for _, n := range n.Nodes {
				visit(n)
			}
		}
	}

	visit(node)
	return fields
}

// validateField checks if a field path exists in the given type
func validateField(typ reflect.Type, field string) error {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	_, found := typ.FieldByName(field)
	if !found {
		return fmt.Errorf("field '%s' not found in type %s", field, typ.Name())
	}
	return nil
}
