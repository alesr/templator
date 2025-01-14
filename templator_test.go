package templator

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Content}}</h1>
</body>
</html>`

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		fs                fstest.MapFS
		opts              []Option[any]
		expectedTemplates []string
		wantErr           bool
	}{
		{
			name: "successful initialization with default options",
			fs: fstest.MapFS{
				"templates/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
				"templates/test.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts: nil,
			expectedTemplates: []string{
				"templates/template1.html",
				"templates/test.html",
			},
			wantErr: false,
		},
		{
			name: "successful initialization with custom path",
			fs: fstest.MapFS{
				"custom/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
				"custom/test.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts: []Option[any]{WithTemplatesPath[any]("custom")},
			expectedTemplates: []string{
				"custom/template1.html",
				"custom/test.html",
			},
			wantErr: false,
		},
		{
			name: "successful initialization with a non-html file",
			fs: fstest.MapFS{
				"templates/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
				"templates/test.txt": &fstest.MapFile{
					Data: []byte("test"),
				},
			},
			opts: nil,
			expectedTemplates: []string{
				"templates/template1.html",
			},
			wantErr: false,
		},
		{
			name: "error loading templates",
			fs: fstest.MapFS{
				"templates/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
				"templates/test.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts:              []Option[any]{WithTemplatesPath[any]("non-existent")},
			expectedTemplates: nil,
			wantErr:           true,
		},
		{
			name: "successful initialization with nested templates",
			fs: fstest.MapFS{
				"templates/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
				"templates/nested/template2.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts: nil,
			expectedTemplates: []string{
				"templates/template1.html",
				"templates/nested/template2.html",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := New(tt.fs, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Len(t, got.templates, len(tt.expectedTemplates))

			for _, tmpl := range tt.expectedTemplates {
				assert.Contains(t, got.templates, tmpl)
			}
		})
	}
}
