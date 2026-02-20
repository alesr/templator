package templator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestData struct {
	Title   string
	Content string
}

const testHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Content}}</h1>
</body>
</html>`

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fs      fstest.MapFS
		opts    []Option[TestData]
		wantErr bool
	}{
		{
			name: "successful initialization with default options",
			fs: fstest.MapFS{
				"templates/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts:    nil,
			wantErr: false,
		},
		{
			name: "successful initialization with custom path",
			fs: fstest.MapFS{
				"custom/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts:    []Option[TestData]{WithTemplatesPath[TestData]("custom")},
			wantErr: false,
		},
		{
			name:    "error with non-existent directory",
			fs:      fstest.MapFS{},
			opts:    []Option[TestData]{WithTemplatesPath[TestData]("non-existent")},
			wantErr: false, // Registry creation succeeds, template loading happens later
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewRegistry(tt.fs, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("can get templates concurrently", func(t *testing.T) {
		fs := fstest.MapFS{
			"custom/template1.html": &fstest.MapFile{
				Data: []byte(testHTMLTemplate),
			},
		}

		opts := []Option[TestData]{WithTemplatesPath[TestData]("custom")}

		registry, err := NewRegistry(fs, opts...)
		require.NoError(t, err)

		var wg sync.WaitGroup
		errs := make(chan error, 2)

		wg.Add(2)
		for range 2 {
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				_, err := registry.Get("template1")
				errs <- err
			}(&wg)
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			require.NoError(t, err)
		}
	})
}

func TestHandler(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/test.html": &fstest.MapFile{
			Data: []byte(testHTMLTemplate),
		},
	}

	reg, err := NewRegistry[TestData](fs)
	require.NoError(t, err)

	tests := []struct {
		name    string
		data    TestData
		wantErr bool
	}{
		{
			name: "successful template execution",
			data: TestData{
				Title:   "Test Title",
				Content: "Test Content",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler, err := reg.Get("test")
			require.NoError(t, err)

			var buf bytes.Buffer
			err = handler.Execute(context.TODO(), &buf, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, buf.String(), tt.data.Title)
			assert.Contains(t, buf.String(), tt.data.Content)
		})
	}
}

type cancelOnFirstWriteWriter struct {
	w        io.Writer
	cancel   context.CancelFunc
	canceled bool
}

func (w *cancelOnFirstWriteWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	if !w.canceled {
		w.canceled = true
		w.cancel()
	}
	return n, err
}

type cancelAndFailWriter struct {
	cancel context.CancelFunc
}

func (w cancelAndFailWriter) Write(_ []byte) (int, error) {
	w.cancel()
	return 0, io.ErrClosedPipe
}

func TestHandler_ExecuteContext(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/test.html": &fstest.MapFile{
			Data: []byte(testHTMLTemplate),
		},
	}

	reg, err := NewRegistry[TestData](fs)
	require.NoError(t, err)

	handler, err := reg.Get("test")
	require.NoError(t, err)

	t.Run("returns cancellation error when context already canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var buf bytes.Buffer
		err := handler.Execute(ctx, &buf, TestData{Title: "Title", Content: "Content"})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)

		var execErr ErrTemplateExecution
		require.ErrorAs(t, err, &execErr)
	})

	t.Run("returns error for nil context", func(t *testing.T) {
		t.Parallel()

		var nilCtx context.Context
		var buf bytes.Buffer
		err := handler.Execute(nilCtx, &buf, TestData{Title: "Title", Content: "Content"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNilContext))

		var execErr ErrTemplateExecution
		require.ErrorAs(t, err, &execErr)
	})

	t.Run("returns cancellation error when canceled during write", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var buf bytes.Buffer
		writer := &cancelOnFirstWriteWriter{w: &buf, cancel: cancel}

		err := handler.Execute(ctx, writer, TestData{Title: "Title", Content: "Content"})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)

		var execErr ErrTemplateExecution
		require.ErrorAs(t, err, &execErr)
	})

	t.Run("returns deadline exceeded when context deadline has passed", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()

		var buf bytes.Buffer
		err := handler.Execute(ctx, &buf, TestData{Title: "Title", Content: "Content"})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)

		var execErr ErrTemplateExecution
		require.ErrorAs(t, err, &execErr)
	})

	t.Run("prefers context cancellation over writer error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := handler.Execute(ctx, cancelAndFailWriter{cancel: cancel}, TestData{Title: "Title", Content: "Content"})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.NotErrorIs(t, err, io.ErrClosedPipe)

		var execErr ErrTemplateExecution
		require.ErrorAs(t, err, &execErr)
	})

	t.Run("executes normally with active context", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		err := handler.Execute(context.Background(), &buf, TestData{Title: "Title", Content: "Content"})
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Title")
		assert.Contains(t, buf.String(), "Content")
	})
}

func TestGet_Error(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/valid.html": &fstest.MapFile{
			Data: []byte(testHTMLTemplate),
		},
		"templates/invalid.html": &fstest.MapFile{
			Data: []byte("{{.InvalidSyntax}}{{end}}"), // Invalid syntax: missing begin block
		},
		"templates/execution_error.html": &fstest.MapFile{
			Data: []byte("{{.NonexistentField}}"), // Valid syntax but will fail during execution because TestData has no such field
		},
		"templates/nested_error.html": &fstest.MapFile{
			Data: []byte("{{.Title}}{{.Content}}{{.Nested.Field}}"), // Will fail when accessing nested field
		},
	}

	reg, err := NewRegistry[TestData](fs)
	require.NoError(t, err)

	tests := []struct {
		name         string
		template     string
		data         TestData
		wantParseErr bool
		wantExecErr  bool
	}{
		{
			name:         "non-existent template",
			template:     "nonexistent",
			data:         TestData{},
			wantParseErr: true,
			wantExecErr:  false,
		},
		{
			name:         "invalid template syntax",
			template:     "invalid",
			data:         TestData{},
			wantParseErr: true, // Should fail during parsing
			wantExecErr:  false,
		},
		{
			name:         "execution error - nonexistent field",
			template:     "execution_error",
			data:         TestData{Title: "Test", Content: "Test"},
			wantParseErr: false,
			wantExecErr:  true, // Execution fails when accessing NonexistentField
		},
		{
			name:         "execution error - nested field",
			template:     "nested_error",
			data:         TestData{Title: "Test", Content: "Test"},
			wantParseErr: false,
			wantExecErr:  true, // Execution fails when accessing Nested.Field
		},
		{
			name:         "valid template",
			template:     "valid",
			data:         TestData{},
			wantParseErr: false,
			wantExecErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler, err := reg.Get(tt.template)
			if tt.wantParseErr {
				require.Error(t, err, "expected parse error")
				require.Nil(t, handler)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, handler)

			var buf bytes.Buffer
			err = handler.Execute(context.TODO(), &buf, tt.data)
			if tt.wantExecErr {
				require.Error(t, err, "expected execution error")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHandler_WithFuncs(t *testing.T) {
	t.Parallel()

	const templateWithFunc = `{{.Title | upper}}`
	fs := fstest.MapFS{
		"templates/withfunc.html": &fstest.MapFile{
			Data: []byte(templateWithFunc),
		},
	}

	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
	}

	reg, err := NewRegistry[TestData](fs, WithTemplateFuncs[TestData](funcMap))
	require.NoError(t, err)

	handler, err := reg.Get("withfunc")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = handler.Execute(context.TODO(), &buf, TestData{Title: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "HELLO", buf.String())
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/concurrent.html": &fstest.MapFile{
			Data: []byte(testHTMLTemplate),
		},
	}

	reg, err := NewRegistry[TestData](fs)
	require.NoError(t, err)

	var wg sync.WaitGroup
	numGoroutines := 10
	errs := make(chan error, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			handler, err := reg.Get("concurrent")
			if err != nil {
				errs <- err
				return
			}

			var buf bytes.Buffer
			data := TestData{
				Title:   fmt.Sprintf("Title %d", i),
				Content: fmt.Sprintf("Content %d", i),
			}
			err = handler.Execute(context.TODO(), &buf, data)
			if err != nil {
				errs <- err
				return
			}

			if !strings.Contains(buf.String(), data.Title) {
				errs <- fmt.Errorf("output missing title %q", data.Title)
				return
			}

			if !strings.Contains(buf.String(), data.Content) {
				errs <- fmt.Errorf("output missing content %q", data.Content)
				return
			}

			errs <- nil
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
}

func TestRegistry_Options(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fs            fstest.MapFS
		opts          []Option[TestData]
		templateName  string
		expectedPath  string
		shouldSucceed bool
	}{
		{
			name: "custom path option",
			fs: fstest.MapFS{
				"custom/path/test.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts:          []Option[TestData]{WithTemplatesPath[TestData]("custom/path")},
			templateName:  "test",
			expectedPath:  "custom/path",
			shouldSucceed: true,
		},
		{
			name: "empty path falls back to default",
			fs: fstest.MapFS{
				"templates/test.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts:          []Option[TestData]{WithTemplatesPath[TestData]("")},
			templateName:  "test",
			expectedPath:  DefaultTemplateDir,
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg, err := NewRegistry(tt.fs, tt.opts...)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPath, reg.config.path) // Updated from reg.path to reg.config.path

			handler, err := reg.Get(tt.templateName)
			if tt.shouldSucceed {
				require.NoError(t, err)
				require.NotNil(t, handler)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestWithTemplateFuncs(t *testing.T) {
	t.Parallel()

	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}

	reg, err := NewRegistry(fstest.MapFS{}, WithTemplateFuncs[TestData](funcMap))
	require.NoError(t, err)
	require.Equal(t, funcMap, reg.config.funcMap)
}

func TestWithFieldValidation(t *testing.T) {
	t.Parallel()

	type ComplexData struct {
		Title    string
		Content  string
		SubTitle *string
	}

	tests := []struct {
		name        string
		templateStr string
		model       ComplexData
		data        ComplexData
		shouldError bool
		expectedErr string
	}{
		{
			name:        "valid fields",
			templateStr: "{{.Title}} {{.Content}}",
			model:       ComplexData{},
			data:        ComplexData{Title: "Test", Content: "Content"},
			shouldError: false,
		},
		{
			name:        "invalid field usage",
			templateStr: "{{.InvalidField}}",
			model:       ComplexData{},
			data:        ComplexData{},
			shouldError: true,
			expectedErr: "field 'InvalidField' not found",
		},
		{
			name:        "nested pointer field valid",
			templateStr: "{{if .SubTitle}}{{.SubTitle}}{{end}}",
			model:       ComplexData{},
			data:        ComplexData{},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fs := fstest.MapFS{
				"templates/test.html": &fstest.MapFile{
					Data: []byte(tt.templateStr),
				},
			}

			reg, err := NewRegistry[ComplexData](fs, WithFieldValidation(tt.model))
			require.NoError(t, err)
			require.True(t, reg.config.validateFields)
			require.Equal(t, tt.model, reg.config.validationModel)

			handler, err := reg.Get("test")
			if tt.shouldError {
				require.Error(t, err)
				if tt.expectedErr != "" {
					require.Contains(t, err.Error(), tt.expectedErr)
				}
				return
			}

			require.NoError(t, err)
			var buf bytes.Buffer
			err = handler.Execute(context.TODO(), &buf, tt.data)
			require.NoError(t, err)
		})
	}
}
