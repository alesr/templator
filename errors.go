package templator

import "fmt"

// ErrTemplateNotFound is returned when a template cannot be found.
type ErrTemplateNotFound struct {
	Name string
}

func (e ErrTemplateNotFound) Error() string {
	return fmt.Sprintf("template '%s' not found", e.Name)
}

// ErrTemplateExecution is returned when a template fails to execute.
type ErrTemplateExecution struct {
	Name string
	Err  error
}

func (e ErrTemplateExecution) Error() string {
	return fmt.Sprintf("failed to execute template '%s': '%v'", e.Name, e.Err)
}

func (e ErrTemplateExecution) Unwrap() error {
	return e.Err
}
