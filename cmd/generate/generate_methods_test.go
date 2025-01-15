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
	// Do not run this test in parallel since it modifies global state
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
	}{
		{
			name:          "default values",
			args:          []string{"cmd"},
			wantTemplates: "templates",
			wantOutput:    "./templator_methods.go",
		},
		{
			name:          "custom values",
			args:          []string{"cmd", "-templates", "custom/templates", "-out", "custom_output.go"},
			wantTemplates: "custom/templates",
			wantOutput:    "custom_output.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osArgsMu.Lock()
			os.Args = tt.args
			osArgsMu.Unlock()

			cfg := parseFlags()
			assert.Equal(t, tt.wantTemplates, cfg.templateDir)
			assert.Equal(t, tt.wantOutput, cfg.outputFile)
		})
	}
}

func TestLoadTemplateGenerator(t *testing.T) {
	tmpl := loadTemplateGenerator()
	require.NotNil(t, tmpl)

	var buf strings.Builder
	err := tmpl.ExecuteTemplate(&buf, "header", nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "package templator")
}

func TestGenerateMethods(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "templator_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templates := map[string]string{
		"index.html":         "<html></html>",
		"users/profile.html": "<html></html>",
	}

	for path, content := range templates {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	outputFile := filepath.Join(tempDir, "output.go")
	cfg := config{
		templateDir: tempDir,
		outputFile:  outputFile,
	}

	tmpl := loadTemplateGenerator()
	generateMethods(cfg, tmpl)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	generatedCode := string(content)
	expectedMethods := []string{
		"GetIndex",
		"GetUsersProfile",
	}

	for _, method := range expectedMethods {
		assert.Contains(t, generatedCode, method)
	}
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
