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

- **Type-safe template handling**:

```go
// Define your template types
type HomeData struct {
    Title   string
    Message string
}

type AboutData struct {
    Company string
    Year    int
}

// Get type-safe handlers
reg := templator.NewRegistry[HomeData](fs)
home, _ := reg.GetHome()
about, _ := reg.GetAbout()

// ✅ Compiles - types match
home.Execute(ctx, w, HomeData{
    Title: "Welcome",
    Message: "Hello",
})

// ❌ Won't compile - wrong data type
home.Execute(ctx, w, AboutData{
    Company: "Acme",
    Year: 2025,
})
```

- **Template customization**:

```go
// Add template functions
home.WithFuncs(template.FuncMap{
    "upper": strings.ToUpper,
}).Execute(ctx, w, data)
```

- **File system abstraction support**:

```go
// Use embedded files
//go:embed templates/*
var embedFS embed.FS
reg := templator.NewRegistry[HomeData](embedFS)

// Use os filesystem
reg := templator.NewRegistry[HomeData](os.DirFS("./templates"))

// Use in-memory filesystem for testing
fsys := fstest.MapFS{
    "templates/home.html": &fstest.MapFile{
        Data: []byte("<h1>{{.Title}}</h1>"),
    },
}
reg := templator.NewRegistry[HomeData](fsys)
```

- **Concurrent-safe operations**:

```go
reg := templator.NewRegistry[HomeData](fs)
home, _ := reg.GetHome()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        home.Execute(ctx, writer, data) // Thread-safe
    }()
}
```

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

### Template Functions

```go
import "strings"

home.WithFuncs(template.FuncMap{
    "upper": strings.ToUpper,
    "join": strings.Join,
}).Execute(ctx, w, data)
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
    templator.WithTemplatesPath[HomeData]("views"),
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
