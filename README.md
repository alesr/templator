# Templator

[![codecov](https://codecov.io/gh/alesr/templator/graph/badge.svg?token=HiYBMXeIbg)](https://codecov.io/gh/alesr/templator)
[![Go Report Card](https://goreportcard.com/badge/github.com/alesr/templator)](https://goreportcard.com/report/github.com/alesr/templator)
[![Go Reference](https://pkg.go.dev/badge/github.com/alesr/templator.svg)](https://pkg.go.dev/github.com/alesr/templator)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/github/license/alesr/templator?color=blue)](https://github.com/alesr/templator/blob/main/LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/alesr/templator)](https://github.com/alesr/templator/releases/latest)

A type-safe HTML template rendering engine for Go.

## Table of Contents

- [Problem Statement](#problem-statement)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage Examples](#usage-examples)
- [Template Generation](#template-generation)
- [Configuration](#configuration)
- [Development Requirements](#development-requirements)
- [Contributing](#contributing)
- [License](#license)


## Problem Statement

Go's built-in template package lacks type safety, which can lead to runtime errors when template data doesn't match what the template expects. For example:

```go
// Traditional approach with Go templates
tmpl := template.Must(template.ParseFiles("home.html"))

// This compiles but will fail at runtime if the template expects different fields
tmpl.Execute(w, struct {
    WrongField string
    MissingRequired int
}{})

// No compile-time checks for:
// - Missing required fields
// - Wrong field types
// - Typos in field names
```

Templator solves this by providing compile-time type checking for your templates:

```go
// Define your template data type
type HomeData struct {
    Title   string
    Content string
}

// Initialize registry with type parameter
reg, _ := templator.NewRegistry[HomeData](fs)

// Get type-safe handler and execute template
home, _ := reg.Get("home")
home.Execute(ctx, w, HomeData{
    Title: "Welcome",
    Content: "Hello",
})

// Won't compile - wrong data structure
home.Execute(ctx, w, struct{
    WrongField string
}{})
```

## Features

- Type-safe template execution with generics
- Concurrent-safe template management
- Custom template functions support
- Clean and simple API
- HTML escaping by default

## Installation

```bash
go get github.com/alesr/templator@latest
```

Install the code generator binary (optional):

```bash
go install github.com/alesr/templator/cmd/generate@latest
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/alesr/templator"
)

// Define your template data
type HomeData struct {
    Title   string
    Content string
}

func main() {
    // Use the filesystem of your choice
    fs := os.DirFS(".")

    // Initialize registry with your data type
    reg, err := templator.NewRegistry[HomeData](fs)
    if err != nil {
        log.Fatal(err)
    }

    // Get type-safe handler for home template
    home, err := reg.Get("home")
    if err != nil {
        log.Fatal(err)
    }

    // Execute template with proper data
    err = home.Execute(context.Background(), os.Stdout, HomeData{
        Title:   "Welcome",
        Content: "Hello, World!",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Usage Examples

### Type-Safe Templates

```go
// Define different data types for different templates
type HomeData struct {
    Title    string
    Content  string
}

type AboutData struct {
    Company  string
    Year     int
}

// Create registries for different template types
homeReg := templator.NewRegistry[HomeData](fs)
aboutReg := templator.NewRegistry[AboutData](fs)

// Get handlers
home, _ := homeReg.Get("home")
about, _ := aboutReg.Get("about")

// Type safety enforced at compile time
home.Execute(ctx, w, HomeData{...})  // ✅ Compiles
home.Execute(ctx, w, AboutData{...}) // ❌ Compile error
```

### Using Template Functions

```go
// Define your custom functions
funcMap := template.FuncMap{
    "upper": strings.ToUpper,
    "lower": strings.ToLower,
}

// Create registry with functions
reg, err := templator.NewRegistry[PageData](fs, 
    templator.WithTemplateFuncs[PageData](funcMap))

// Use functions in your templates:
// <h1>{{.Title | upper}}</h1>
```

### File System Support

```go
// Embedded FS
//go:embed templates/*
var embedFS embed.FS
reg := templator.NewRegistry[HomeData](embedFS)

// OS File System
reg := templator.NewRegistry[HomeData](os.DirFS("./templates"))

// In-Memory (testing)
fsys := fstest.MapFS{
    "templates/home.html": &fstest.MapFile{
        Data: []byte(`<h1>{{.Title}}</h1>`),
    },
}
reg := templator.NewRegistry[HomeData](fsys)
```

### Field Validation

```go
type ArticleData struct {
    Title    string    // Only these two fields
    Content  string    // are allowed in templates
}

// Enable validation during registry creation
reg := templator.NewRegistry[ArticleData](
    fs,
    templator.WithFieldValidation(ArticleData{}),
)

// Example templates:

// valid.html:
// <h1>{{.Title}}</h1>           // ✅ OK - Title exists in ArticleData
// <p>{{.Content}}</p>           // ✅ OK - Content exists in ArticleData

// invalid.html:
// <h1>{{.Author}}</h1>          // ❌ Error - Author field doesn't exist
// <p>{{.PublishedAt}}</p>       // ❌ Error - PublishedAt field doesn't exist

// Using the templates:
handler, err := reg.Get("valid")    // ✅ Success - all fields exist
if err != nil {
    log.Fatal(err)
}

handler, err := reg.Get("invalid")  // ❌ Error: "template 'invalid' validation error: Author - field 'Author' not found in type ArticleData"

// The validation error provides:
// - Template name
// - Invalid field path
// - Detailed error message
if validErr, ok := err.(*templator.ValidationError); ok {
    fmt.Printf("Template: %s\n", validErr.TemplateName)
    fmt.Printf("Invalid field: %s\n", validErr.FieldPath)
    fmt.Printf("Error: %v\n", validErr.Err)
}
```

This validation happens when loading the template, not during execution, helping catch field mismatches early in development.

## Template Generation

Generate accessors in your package:

```bash
go run github.com/alesr/templator/cmd/generate \
  -package myapp \
  -templates ./templates \
  -out ./templator_accessors_gen.go
```

Then use the generated wrapper:

```go
reg, _ := templator.NewRegistry[HomeData](fs)
tpl := NewTemplateAccessors(reg)
home, _ := tpl.GetHome()
```

Template names are mapped to wrapper methods using title-cased path components:

```zsh
templates/
├── home.html           -> tpl.GetHome()
├── about.html          -> tpl.GetAbout()
└── components/
    └── header.html     -> tpl.GetComponentsHeader()
```

This creates a generated file (for example `templator_accessors_gen.go`) with wrapper methods such as:

```go
func (r *TemplateAccessors[T]) GetHome() (*templator.Handler[T], error) {
    return r.registry.Get("home")
}

func (r *TemplateAccessors[T]) GetAbout() (*templator.Handler[T], error) {
    return r.registry.Get("about")
}
```

This file is automatically generated and should not be manually edited.

## Configuration

```go
reg, err := templator.NewRegistry[HomeData](
    fs,
    // Custom template directory
    templator.WithTemplatesPath[HomeData]("views"),
    // Enable field validation
    templator.WithFieldValidation(HomeData{}),
)
```

## Development Requirements

- Go 1.24 or higher

## License

MIT
