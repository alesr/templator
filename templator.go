// Package templator provides a type-safe template rendering system for Go applications.
// It offers a simple and concurrent-safe way to manage HTML templates with compile-time
// type checking for template data.
package templator

//go:generate go run ./cmd/generate/generate_methods.go

import (
	"context"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"sync"
)

const (
	// DefaultTemplateDir is the default directory where templates are stored.
	DefaultTemplateDir = "templates"
	// DefaultTemplateExt is the default file extension for templates.
	DefaultTemplateExt = "html"

	// ExtensionHTML defines the standard HTML template file extension.
	ExtensionHTML Extension = ".html"
)

// Extension represents a template file extension type.
type Extension string

// Option configures a Registry instance.
type Option[T any] func(*Registry[T])

// WithTemplatesPath returns an Option that sets a custom template directory path.
// If an empty path is provided, the default path will be used.
func WithTemplatesPath[T any](path string) Option[T] {
	return func(r *Registry[T]) {
		if path != "" {
			r.config.path = path
		}
	}
}

// WithFieldValidation enables template field validation against the provided model
func WithFieldValidation[T any](model T) Option[T] {
	return func(r *Registry[T]) {
		r.config.validateFields = true
		r.config.validationModel = model
	}
}

type config[T any] struct {
	path            string
	validateFields  bool
	validationModel T
}

// Registry manages template handlers in a concurrent-safe manner.
type Registry[T any] struct {
	fs     fs.FS
	config config[T]
	mu     sync.RWMutex
}

// Handler manages a specific template instance with type-safe data handling.
// It provides methods for template execution and customization.
type Handler[T any] struct {
	tmpl *template.Template
	reg  *Registry[T]
}

// NewRegistry creates a new template registry with the provided filesystem and options.
// It accepts a filesystem interface and variadic options for customization.
func NewRegistry[T any](fsys fs.FS, opts ...Option[T]) (*Registry[T], error) {
	reg := &Registry[T]{
		fs: fsys,
		config: config[T]{
			path: DefaultTemplateDir,
		},
	}
	for _, opt := range opts {
		opt(reg)
	}
	return reg, nil
}

// Get retrieves or creates a type-safe handler for a specific template.
// It automatically appends the .html extension to the template name.
// Returns an error if the template cannot be parsed.
func (r *Registry[T]) Get(name string) (*Handler[T], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, err := template.ParseFS(r.fs, filepath.Join(r.config.path, name+".html"))
	if err != nil {
		return nil, err
	}

	if r.config.validateFields {
		if err := validateTemplateFields(name, tmpl.Tree, r.config.validationModel); err != nil {
			return nil, err
		}
	}

	return &Handler[T]{
		tmpl: tmpl,
		reg:  r,
	}, nil
}

// Execute renders the template with the provided data and writes the output to the writer.
// The context parameter can be used for cancellation and deadline control.
func (h *Handler[T]) Execute(ctx context.Context, w io.Writer, data T) error {
	return h.tmpl.Execute(w, data)
}

// WithFuncs adds custom template functions to the handler.
// Returns the handler for method chaining.
func (h *Handler[T]) WithFuncs(funcMap template.FuncMap) *Handler[T] {
	h.tmpl = h.tmpl.Funcs(funcMap)
	return h
}
