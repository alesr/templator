package templator

//go:generate go run ./cmd/generate/generate_methods.go

import (
	"fmt"
	"html/template"
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

type Templator[T any] struct {
	templates map[string]*template.Template
	fs        fs.FS
	path      string
	ext       string
	mu        sync.RWMutex
}

// Option type for customizations
type Option[T any] func(*Templator[T])

func WithTemplatesPath[T any](path string) Option[T] {
	return func(t *Templator[T]) {
		if path == "" {
			return
		}
		t.path = path
	}
}

func New[T any](fsys fs.FS, opts ...Option[T]) (*Templator[T], error) {
	t := &Templator[T]{
		templates: make(map[string]*template.Template),
		fs:        fsys,
		path:      DefaultTemplateDir,
		ext:       "." + DefaultTemplateExt, // Add dot prefix
	}

	for _, opt := range opts {
		opt(t)
	}

	if err := t.loadTemplates(); err != nil {
		return nil, fmt.Errorf("error loading templates: %w", err)
	}
	return t, nil
}

func (t *Templator[T]) loadTemplates() error {
	return fs.WalkDir(t.fs, t.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Skip directories
			return nil
		}

		ext := filepath.Ext(path)
		if ext != t.ext {
			return nil
		}

		tmpl, err := template.ParseFS(t.fs, path)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", path, err)
		}

		t.mu.Lock()
		t.templates[path] = tmpl
		t.mu.Unlock()

		return nil
	})
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
