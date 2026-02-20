// Package templator provides a type-safe template rendering system for Go applications.
// It offers a simple and concurrent-safe way to manage HTML templates with compile-time
// type checking for template data.
package templator

//go:generate go run ./cmd/generate/generate_methods.go

import (
	"context"
	"errors"
	"html/template"
	"io"
	"io/fs"
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

func WithTemplateFuncs[T any](funcMap template.FuncMap) Option[T] {
	return func(r *Registry[T]) {
		r.config.funcMap = funcMap
	}
}

type config[T any] struct {
	path            string
	validateFields  bool
	validationModel T
	funcMap         template.FuncMap
}

// Registry manages template handlers in a concurrent-safe manner.
type Registry[T any] struct {
	fs        fs.FS
	config    config[T]
	mu        sync.RWMutex
	templates map[string]*Handler[T]
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
		templates: make(map[string]*Handler[T]),
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
	if h, ok := r.templates[name]; ok {
		r.mu.RUnlock()
		return h, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check again in case another goroutine has loaded the template
	if h, ok := r.templates[name]; ok {
		return h, nil
	}

	// Read template content first
	content, err := fs.ReadFile(r.fs, r.config.path+"/"+name+".html")
	if err != nil {
		return nil, err
	}

	// Validate fields if enabled - validate content before parsing
	if r.config.validateFields {
		if err := validateTemplateFields(name, string(content), r.config.validationModel); err != nil {
			return nil, err
		}
	}

	// Parse template after validation
	tmpl, err := template.New(name + ".html").Funcs(r.config.funcMap).Parse(string(content))
	if err != nil {
		return nil, err
	}

	handler := &Handler[T]{tmpl: tmpl, reg: r}
	r.templates[name] = handler
	return handler, nil
}

var ErrNilContext = errors.New("nil context")

// Execute renders the template with the provided data and writes the output to the writer.
// Context cancellation and deadlines are checked before rendering, on each write,
// and after rendering. Cancellation is best-effort at write boundaries.
func (h *Handler[T]) Execute(ctx context.Context, w io.Writer, data T) error {
	if ctx == nil {
		return ErrTemplateExecution{Name: h.tmpl.Name(), Err: ErrNilContext}
	}

	wrappedWriter := contextWriter{Writer: w, ctx: ctx}

	if err := h.tmpl.Execute(wrappedWriter, data); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ErrTemplateExecution{Name: h.tmpl.Name(), Err: ctxErr}
		}
		return ErrTemplateExecution{Name: h.tmpl.Name(), Err: err}
	}

	if err := ctx.Err(); err != nil {
		return ErrTemplateExecution{Name: h.tmpl.Name(), Err: err}
	}
	return nil
}

type contextWriter struct {
	io.Writer
	ctx context.Context
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.Writer.Write(p)
}
