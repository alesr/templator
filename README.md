# Templator

A type-safe HTML template rendering engine for Go.

## Features

- **Type-safe template execution**: Go generics ensure your template data matches at compile time:

  ```go
  // Define your template types
  type HomeData struct {
      Title string
      Message string
  }

  type AboutData struct {
      Company string
      Year int
  }

  // The compiler enforces correct data types
  tmpl, _ := templator.New[HomeData](fs)
  
  // ✅ This compiles - types match
  tmpl.ExecuteHome(ctx, w, HomeData{
      Title: "Welcome",
      Message: "Hello",
  })

  // ❌ This won't compile - wrong data type
  tmpl.ExecuteHome(ctx, w, AboutData{
      Company: "Acme",
      Year: 2024,
  })
  ```

- **File system abstraction support**: Use any filesystem implementation:

  ```go
  // Use embedded files
  //go:embed templates/*
  var embedFS embed.FS
  tmpl, _ := templator.New[HomeData](embedFS)

  // Use os filesystem
  tmpl, _ := templator.New[HomeData](os.DirFS("./templates"))

  // Use in-memory filesystem for testing
  fsys := fstest.MapFS{
      "templates/home.html": &fstest.MapFile{
          Data: []byte("<h1>{{.Title}}</h1>"),
      },
  }
  tmpl, _ := templator.New[HomeData](fsys)
  ```

- **Concurrent-safe operations**: Safe for concurrent template execution:
  
  ```go
  // Safe to use from multiple goroutines
  tmpl, _ := templator.New[HomeData](fs)
  
  var wg sync.WaitGroup
  for i := 0; i < 100; i++ {
      wg.Add(1)
      go func() {
          defer wg.Done()
          tmpl.ExecuteHome(ctx, writer, data) // Thread-safe
      }()
  }
  ```

- **Automatic template loading**: Templates are discovered and parsed automatically:

  ```go
  // Directory structure:
  // templates/
  //   ├── home.html
  //   ├── about.html
  //   └── components/
  //       └── header.html
  
  // All templates are loaded and parsed automatically
  tmpl, _ := templator.New[HomeData](fs)
  // Generated methods: ExecuteHome, ExecuteAbout, ExecuteComponentsHeader
  ```

- **Custom template directory support**: Flexible template organization:

  ```go
  // Default templates directory
  tmpl, _ := templator.New[HomeData](fs)
  
  // Custom directory
  tmpl, _ := templator.New[HomeData](
      fs,
      templator.WithTemplatesPath[HomeData]("views"),
  )
  ```

## Installation

```go
go install github.com/alesr/templator
```

## Usage

```go
import "github.com/alesr/templator"

// Define your template data type
type HomeData struct {
    Title   string
    Content string
}

// Initialize templator with the embedded file system
tmpl, err := templator.New[HomeData](embedFS)
if err != nil {
    log.Fatal(err)
}

// Execute the template
data := HomeData{
    Title:   "Welcome",
    Content: "Hello World",
}

if err := tmpl.ExecuteHome(ctx, w, data); err != nil {
    log.Fatal(err)
}
```

## Template Method Generation

Templator uses Go's `go:generate` to automatically create type-safe template execution methods. Each HTML template gets its own strongly-typed method.

### How it works

1. Place your HTML templates in the `templates` directory
2. Run `go generate` to create type-safe methods
3. The generator supports these flags:

   ```bash
   # Default values shown
   go run cmd/generate/generate_methods.go \
     -templates="templates"    # Source templates directory
     -out="templator_methods.go"    # Output file location
   ```

4. Type-safe methods are generated for each template, following the pattern:
   - `templates/home.html` → `ExecuteHome` method
   - `templates/user/profile.html` → `ExecuteUserProfile` method

Example:

- Template: `templates/home.html`
- Generated:

  ```go
  type HomeData struct {
      Title string
      Content string
  }
  
  func (t *Templator[HomeData]) ExecuteHome(ctx context.Context, w io.Writer, data HomeData) error
  ```

### Directory Structure

## Options

```go
// Custom template directory
tmpl, err := templator.New[HomeData](
    embedFS, 
    templator.WithTemplatesPath[HomeData]("custom/path"),
)
```

## License

MIT
