package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var osArgsMu sync.Mutex

func TestParseFlags(t *testing.T) {
	osArgsMu.Lock()
	oldArgs := os.Args
	osArgsMu.Unlock()

	defer func() {
		osArgsMu.Lock()
		os.Args = oldArgs
		osArgsMu.Unlock()
	}()

	tests := []struct {
		name          string
		args          []string
		wantTemplates string
		wantOutput    string
		wantPackage   string
		wantErr       string
	}{
		{
			name:          "default values",
			args:          []string{"cmd"},
			wantTemplates: "templates",
			wantOutput:    "./templator_accessors_gen.go",
			wantPackage:   "main",
		},
		{
			name:          "custom values",
			args:          []string{"cmd", "-templates", "custom/templates", "-out", "custom_output.go"},
			wantTemplates: "custom/templates",
			wantOutput:    "custom_output.go",
			wantPackage:   "main",
		},
		{
			name:          "custom package",
			args:          []string{"cmd", "-package", "myapp"},
			wantTemplates: "templates",
			wantOutput:    "./templator_accessors_gen.go",
			wantPackage:   "myapp",
		},
		{
			name:    "rejects empty package",
			args:    []string{"cmd", "-package", ""},
			wantErr: "requires non-empty -package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osArgsMu.Lock()
			os.Args = tt.args
			osArgsMu.Unlock()

			cfg, err := parseFlags()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTemplates, cfg.templateDir)
			assert.Equal(t, tt.wantOutput, cfg.outputFile)
			assert.Equal(t, tt.wantPackage, cfg.packageName)
		})
	}
}

func TestLoadTemplateGenerator(t *testing.T) {
	tmpl, err := loadTemplateGenerator()
	require.NoError(t, err)
	require.NotNil(t, tmpl)

	var buf strings.Builder
	err = tmpl.ExecuteTemplate(&buf, "header", headerData{PackageName: "templator"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "package templator")
}

func TestGenerateMethods(t *testing.T) {
	tempDir := t.TempDir()

	writeTemplateFixture(t, tempDir, "index.html")
	writeTemplateFixture(t, tempDir, "users/profile.html")

	outputFile := filepath.Join(tempDir, "output.go")
	cfg := config{
		templateDir:     tempDir,
		outputFile:      outputFile,
		packageName:     "myapp",
		templatorImport: "github.com/alesr/templator",
	}

	tmpl, err := loadTemplateGenerator()
	require.NoError(t, err)

	err = generateMethods(cfg, tmpl)
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedCode := string(content)
	assert.Contains(t, generatedCode, "package myapp")
	assert.Contains(t, generatedCode, "import \"github.com/alesr/templator\"")
	assert.Contains(t, generatedCode, "type TemplateAccessors[T any] struct")
	assert.Contains(t, generatedCode, "func NewTemplateAccessors[T any](registry *templator.Registry[T]) *TemplateAccessors[T]")
	assert.Contains(t, generatedCode, "func (r *TemplateAccessors[T]) GetIndex() (*templator.Handler[T], error)")
	assert.Contains(t, generatedCode, "func (r *TemplateAccessors[T]) GetUsersProfile() (*templator.Handler[T], error)")
}

func TestBuildTemplateData(t *testing.T) {
	tests := []struct {
		name     string
		relPath  string
		wantName string
		wantPath string
	}{
		{
			name:     "simple template",
			relPath:  "index.html",
			wantName: "GetIndex",
			wantPath: "index",
		},
		{
			name:     "nested template",
			relPath:  "users/profile.html",
			wantName: "GetUsersProfile",
			wantPath: "users/profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := buildTemplateData(tt.relPath, cases.Title(language.English))
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, data.MethodName)
			assert.Equal(t, tt.wantPath, data.TemplateName)
		})
	}
}

func writeTemplateFixture(t *testing.T, rootDir, relativePath string) {
	t.Helper()

	fullPath := filepath.Join(rootDir, relativePath)
	err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(fullPath, []byte("<html></html>"), 0o644)
	require.NoError(t, err)
}
