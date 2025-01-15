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
	DefaultTemplateDir = "templates"
	DefaultTemplateExt = "html"

	// ExtensionHTML is the extension for HTML templates.
	ExtensionHTML Extension = ".html"
)

type Extension string

// Option configures a Registry
type Option[T any] func(*Registry[T])

// WithTemplatesPath sets a custom template directory path
func WithTemplatesPath[T any](path string) Option[T] {
	return func(r *Registry[T]) {
		if path != "" {
			r.path = path
		}
	}
}

// Registry manages all template handlers
type Registry[T any] struct {
	fs   fs.FS
	path string
	mu   sync.RWMutex
}

// Handler manages a specific template
type Handler[T any] struct {
	tmpl *template.Template
	reg  *Registry[T]
}

func NewRegistry[T any](fsys fs.FS, opts ...Option[T]) (*Registry[T], error) {
	reg := &Registry[T]{
		fs:   fsys,
		path: DefaultTemplateDir,
	}

	for _, opt := range opts {
		opt(reg)
	}
	return reg, nil
}

// Get returns a type-safe handler for a specific template
func (r *Registry[T]) Get(name string) (*Handler[T], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, err := template.ParseFS(r.fs, filepath.Join(r.path, name+".html"))
	if err != nil {
		return nil, err
	}

	return &Handler[T]{
		tmpl: tmpl,
		reg:  r,
	}, nil
}

// Usage example:
// handler := reg.Get[HomeData]("home")

// Execute executes the template with type-safe data
func (h *Handler[T]) Execute(ctx context.Context, w io.Writer, data T) error {
	return h.tmpl.Execute(w, data)
}

// WithFuncs adds template functions
func (h *Handler[T]) WithFuncs(funcMap template.FuncMap) *Handler[T] {
	h.tmpl = h.tmpl.Funcs(funcMap)
	return h
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
