package templator

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

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

			got, err := NewRegistry[TestData](tt.fs, tt.opts...)
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

		wg.Add(2)
		for i := 0; i < 2; i++ {
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				_, err := registry.Get("template1")
				require.NoError(t, err)
			}(&wg)
		}
		wg.Wait()
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
			err = handler.Execute(context.Background(), &buf, tt.data)

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
			err = handler.Execute(context.Background(), &buf, tt.data)
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
	err = handler.Execute(context.Background(), &buf, TestData{Title: "hello"})
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

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			handler, err := reg.Get("concurrent")
			require.NoError(t, err)

			var buf bytes.Buffer
			data := TestData{
				Title:   fmt.Sprintf("Title %d", i),
				Content: fmt.Sprintf("Content %d", i),
			}
			err = handler.Execute(context.Background(), &buf, data)
			require.NoError(t, err)
			assert.Contains(t, buf.String(), data.Title)
			assert.Contains(t, buf.String(), data.Content)
		}(i)
	}

	wg.Wait()
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

			reg, err := NewRegistry[TestData](tt.fs, tt.opts...)
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

	reg, err := NewRegistry[TestData](fstest.MapFS{}, WithTemplateFuncs[TestData](funcMap))
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
			err = handler.Execute(context.Background(), &buf, tt.data)
			require.NoError(t, err)
		})
	}
}
