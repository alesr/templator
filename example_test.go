package templator_test

import (
	"bytes"
	"context"
	"fmt"
	"testing/fstest"

	"github.com/alesr/templator"
)

func Example() {
	// Create a virtual filesystem for testing
	fs := fstest.MapFS{
		"templates/home.html": &fstest.MapFile{
			Data: []byte(`<h1>{{.Title}}</h1><p>{{.Content}}</p>`),
		},
		"templates/about/team.html": &fstest.MapFile{
			Data: []byte(`<h2>{{.TeamName}}</h2><p>{{.Description}}</p>`),
		},
	}

	// Example 1: Basic template with HomeData
	type HomeData struct {
		Title   string
		Content string
	}

	homeReg, _ := templator.NewRegistry[HomeData](fs)
	home, _ := homeReg.Get("home")

	var buf bytes.Buffer
	home.Execute(context.Background(), &buf, HomeData{
		Title:   "Welcome",
		Content: "Hello, World!",
	})

	fmt.Println("Home template output:")
	fmt.Println(buf.String())

	// Example 2: Template with custom path and different data type
	type TeamData struct {
		TeamName    string
		Description string
	}

	teamReg, _ := templator.NewRegistry[TeamData](fs)
	team, _ := teamReg.Get("about/team")

	buf.Reset()
	team.Execute(context.Background(), &buf, TeamData{
		TeamName:    "Engineering",
		Description: "Building amazing things",
	})

	fmt.Println("\nTeam template output:")
	fmt.Println(buf.String())

	// Output:
	// Home template output:
	// <h1>Welcome</h1><p>Hello, World!</p>
	//
	// Team template output:
	// <h2>Engineering</h2><p>Building amazing things</p>
}
