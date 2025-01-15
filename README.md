# Templator

[![codecov](https://codecov.io/gh/alesr/templator/graph/badge.svg?token=HiYBMXeIbg)](https://codecov.io/gh/alesr/templator)

A type-safe HTML template rendering engine for Go.

> **Note**: This is an experimental project in active development. While it's stable enough for use, expect possible API changes. Feedback and contributions are welcome!

## Table of Contents

- [Problem Statement](#problem-statement)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage Examples](#usage-examples)
- [Template Generation](#template-generation)
- [Configuration](#configuration)
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
home, _ := reg.GetHome()
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
go install github.com/alesr/templator
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
    home, err := reg.GetHome()
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
home, _ := homeReg.GetHome()
about, _ := aboutReg.GetAbout()

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
```

This validation happens when loading the template, not during execution, helping catch field mismatches early in development.

## Template Generation

Templates are automatically discovered and type-safe methods are generated:

```zsh
templates/
├── home.html           -> reg.GetHome()
├── about.html          -> reg.GetAbout()
└── components/
    └── header.html     -> reg.GetComponentsHeader()

# Generate methods
go generate ./...
```

The generation process creates a `templator_methods.go` file containing type-safe method handlers for each template. For example:

```go
// Code generated by go generate; DO NOT EDIT.
func (r *Registry[T]) GetHome() (*Handler[T], error) {
    return r.Get("home")
}

func (r *Registry[T]) GetAbout() (*Handler[T], error) {
    return r.Get("about")
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

## Contributing

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)

3. Make your changes
4. Run tests (`go test ./...`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Requirements

- Go 1.21 or higher

## License

MIT
