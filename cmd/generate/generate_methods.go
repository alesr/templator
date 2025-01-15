package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/alesr/templator"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type TemplateData struct {
	DataType     string
	MethodName   string
	TemplateName string
}

func main() {
	cfg := parseFlags()
	tmpl := loadTemplateGenerator()
	generateMethods(cfg, tmpl)
}

type config struct {
	templateDir string
	outputFile  string
}

func parseFlags() config {
	templateDir := flag.String(
		"templates",
		templator.DefaultTemplateDir,
		"directory containing the template files",
	)

	outputFile := flag.String(
		"out",
		"./templator_methods.go",
		"output file for generated methods",
	)

	flag.Parse()
	return config{*templateDir, *outputFile}
}

func loadTemplateGenerator() *template.Template {
	tmpl, err := template.ParseFiles("cmd/generate/templates/templates.tmpl")
	if err != nil {
		panic(fmt.Errorf("failed to parse code generator template: %w", err))
	}
	return tmpl
}

func generateMethods(cfg config, tmpl *template.Template) {
	var buf bytes.Buffer
	if err := writeHeader(&buf, tmpl); err != nil {
		panic(fmt.Errorf("failed to write header: %w", err))
	}

	if err := processTemplates(cfg.templateDir, &buf, tmpl); err != nil {
		panic(fmt.Errorf("failed to process templates: %w", err))
	}

	if err := writeOutput(cfg.outputFile, &buf); err != nil {
		panic(fmt.Errorf("failed to write output: %w", err))
	}
	fmt.Println("Methods generated and formatted successfully.")
}

func writeHeader(buf *bytes.Buffer, tmpl *template.Template) error {
	return tmpl.ExecuteTemplate(buf, "header", nil)
}

func processTemplates(templateDir string, buf *bytes.Buffer, tmpl *template.Template) error {
	caser := cases.Title(language.English)

	return filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if filepath.Ext(path) != string(templator.ExtensionHTML) {
			return nil
		}
		return generateTemplateMethod(path, templateDir, buf, tmpl, caser)
	})
}

func generateTemplateMethod(path, templateDir string, buf *bytes.Buffer, tmpl *template.Template, caser cases.Caser) error {
	relPath, err := filepath.Rel(templateDir, path)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	data, err := buildTemplateData(relPath, caser)
	if err != nil {
		return fmt.Errorf("failed to build template data: %w", err)
	}

	if err := tmpl.ExecuteTemplate(buf, "method", data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	buf.WriteString("\n")
	return nil
}

func buildTemplateData(relPath string, caser cases.Caser) (TemplateData, error) {
	basePath := strings.TrimSuffix(relPath, string(templator.ExtensionHTML))

	parts := strings.Split(basePath, string(filepath.Separator))
	for i, part := range parts {
		parts[i] = caser.String(part)
	}

	return TemplateData{
		MethodName:   "Get" + strings.Join(parts, ""),
		TemplateName: filepath.ToSlash(basePath),
	}, nil
}

func writeOutput(outputFile string, buf *bytes.Buffer) error {
	formattedSource, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}
	return os.WriteFile(outputFile, formattedSource, 0644)
}
