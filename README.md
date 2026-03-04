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
- [What Templator Solves](#what-templator-solves)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [How It All Works Together](#how-it-all-works-together)
- [Usage Examples](#usage-examples)
- [Template Generation](#template-generation)
- [Configuration](#configuration)
- [Development Requirements](#development-requirements)
- [Contributing](#contributing)
- [License](#license)

## Problem Statement

Go's built-in `html/template` package is great, but it has one big weakness: no type safety.

```go
tmpl := template.Must(template.ParseFiles("home.html"))

// this compiles fine...
err := tmpl.Execute(w, struct {
    WrongField   string
    MissingStuff int
}{})
```

You only find out at runtime that your data does not match what the template expects: missing fields, typos, wrong types, and so on.

## What Templator Solves

Templator wraps `html/template` and gives you compile-time guarantees with generics.

You define your data once:

```go
type HomeData struct {
    Title   string
    Content string
}
```

Then template execution is type-checked against the template by the compiler.

You can get a handler in two equivalent ways:

1. Simple and direct: `reg.Get("home")`
2. IDE-friendly codegen: `tpl.GetHome()`

Both return `*Handler[T]` with the same behavior. The difference is only developer experience.

## Features

- Compile-time type safety via generics
- Optional field validation (catches mismatches when loading templates)
- Concurrent-safe template management with `fs.FS` support
- Custom template functions
- Context cancellation and deadline propagation
- Minimal overhead on top of `html/template`
- Small API with optional code generation helpers

## Installation

```bash
go get github.com/alesr/templator@latest
```

Optional generator binary:

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

type HomeData struct {
    Title   string
    Content string
}

func main() {
    fs := os.DirFS(".")

    reg, err := templator.NewRegistry[HomeData](fs)
    if err != nil {
        log.Fatal(err)
    }

    home, err := reg.Get("home")
    if err != nil {
        log.Fatal(err)
    }

    err = home.Execute(context.Background(), os.Stdout, HomeData{
        Title:   "Welcome",
        Content: "Hello, World!",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## How It All Works Together

1. Define a data struct (this is your type parameter `T`).
2. Create a `Registry[T]` for your filesystem.
3. Get a `Handler[T]` either with `reg.Get("name")` or generated accessors like `tpl.GetName()`.
4. Execute with `handler.Execute(ctx, writer, data)`.
5. Optionally enable field validation, custom functions, or a different templates path.

Everything is still powered by the standard `html/template` package.

## Usage Examples

### Type-Safe Templates (different data per template)

```go
type HomeData struct{ Title, Content string }
type AboutData struct{ Company string; Year int }

homeReg, _ := templator.NewRegistry[HomeData](fs)
aboutReg, _ := templator.NewRegistry[AboutData](fs)

home, _ := homeReg.Get("home")
about, _ := aboutReg.Get("about")

home.Execute(ctx, w, HomeData{...})  // ok
home.Execute(ctx, w, AboutData{...}) // trying to pass about data to the home tmpl is caught by the compiler
```

### Custom Template Functions

```go
funcMap := template.FuncMap{
    "upper": strings.ToUpper,
}

reg, _ := templator.NewRegistry[PageData](
    fs,
    templator.WithTemplateFuncs[PageData](funcMap),
)
```

Use in templates as usual: `{{.Title | upper}}`

### File System Support

```go
// Embedded FS
//go:embed templates/*
var embedFS embed.FS

reg, _ := templator.NewRegistry[HomeData](embedFS)

// os dir
reg, _ = templator.NewRegistry[HomeData](os.DirFS("templates"))

// in-memory for tests
fsys := fstest.MapFS{ /* ... */ }
reg, _ = templator.NewRegistry[HomeData](fsys)
```

### Field Validation (catches errors early)

```go
type ArticleData struct {
    Title   string
    Content string
}

reg, _ := templator.NewRegistry[ArticleData](
    fs,
    templator.WithFieldValidation(ArticleData{}),
)
```

If a template references `{{.Author}}` but `Author` does not exist in `ArticleData`, `Get(...)` returns a validation error.

## Template Generation

Want `tpl.GetHome()` instead of string lookup? Use the generator.

```bash
go run github.com/alesr/templator/cmd/generate \
  -package main \
  -templates ./templates \
  -out ./templator_accessors_gen.go
```

You can also wire it into `go generate`:

```go
//go:generate go run github.com/alesr/templator/cmd/generate -package main -templates ./templates -out ./templator_accessors_gen.go
```

Then use the generated wrapper:

```go
reg, _ := templator.NewRegistry[HomeData](fs)
tpl := NewTemplateAccessors(reg)

home, _ := tpl.GetHome()
header, _ := tpl.GetComponentsHeader()
```

`components/header.html` maps to `GetComponentsHeader`.

For full generator docs (flags, behavior, and examples), see [`cmd/generate/README.md`](cmd/generate/README.md).

## Configuration

```go
reg, err := templator.NewRegistry[HomeData](
    fs,
    templator.WithTemplatesPath[HomeData]("views"),
    templator.WithTemplateFuncs[HomeData](funcMap),
    templator.WithFieldValidation(HomeData{}),
)
```

## Development Requirements

- Go 1.24 or higher

## Contributing

Contributions are welcome.

## License

MIT
