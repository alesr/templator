package templator_test

import (
	"bytes"
	"context"
	"fmt"
	"testing/fstest"

	"github.com/alesr/templator"
)

// TestData represents the structure used for template data
type TestData struct {
	Title   string
	Content string
}

func Example() {
	// Create template files in memory
	fs := fstest.MapFS{
		"templates/home.html": &fstest.MapFile{
			Data: []byte(`<h1>{{.Title}}</h1><p>{{.Content}}</p>`),
		},
		"templates/team.html": &fstest.MapFile{
			Data: []byte(`<h2>{{.Title}}</h2><p>{{.Content}}</p>`),
		},
	}

	// Create a new registry
	reg, _ := templator.NewRegistry[TestData](fs)

	// Get and execute home template
	home, _ := reg.Get("home")
	var homeOutput bytes.Buffer
	home.Execute(context.Background(), &homeOutput, TestData{
		Title:   "Welcome",
		Content: "Hello, World!",
	})

	// Get and execute team template
	team, _ := reg.Get("team")
	var teamOutput bytes.Buffer
	team.Execute(context.Background(), &teamOutput, TestData{
		Title:   "Engineering",
		Content: "Building amazing things",
	})

	// Print the outputs
	fmt.Printf("Home template output:\n%s\n\n", homeOutput.String())
	fmt.Printf("Team template output:\n%s\n", teamOutput.String())

	// Output:
	// Home template output:
	// <h1>Welcome</h1><p>Hello, World!</p>
	//
	// Team template output:
	// <h2>Engineering</h2><p>Building amazing things</p>
}
