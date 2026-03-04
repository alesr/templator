# Template Accessor Generator

This command generates typed accessor methods for templates in your application package.

Instead of calling `reg.Get("home")` with string names everywhere, you can generate methods like `tpl.GetHome()` and `tpl.GetComponentsHeader()`.

## What it generates

Given templates like:

```text
templates/
  home.html
  about.html
  components/header.html
```

The generator writes a Go file (for example `templator_accessors_gen.go`) with:

- `TemplateAccessors[T]`
- `NewTemplateAccessors(...)`
- One method per template (`GetHome`, `GetAbout`, `GetComponentsHeader`, ...)

These methods call `registry.Get("...")` under the hood.

## Run it

From your app module, run:

```bash
go run github.com/alesr/templator/cmd/generate \
  -package main \
  -templates ./templates \
  -out ./templator_accessors_gen.go
```

## Flags

- `-templates` (default: `templates`): directory to scan for `.html` templates
- `-out` (default: `./templator_accessors_gen.go`): output file path
- `-package` (default: `main`): package name for generated code
- `-templator-import` (default: `github.com/alesr/templator`): import path used in generated file

## Use with go:generate

Add this line in one of your source files:

```go
//go:generate go run github.com/alesr/templator/cmd/generate -package main -templates ./templates -out ./templator_accessors_gen.go
```

Then run:

```bash
go generate ./...
```

## Example usage in app code

```go
reg, _ := templator.NewRegistry[HomeData](fs)
tpl := NewTemplateAccessors(reg)

home, _ := tpl.GetHome()
header, _ := tpl.GetComponentsHeader()
```

## Notes

- Re-run generation whenever templates are added, removed, or renamed.
- Generated methods are optional convenience APIs; `reg.Get("name")` always remains available.
