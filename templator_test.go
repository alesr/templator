package templator

import (
	"bytes"
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestData struct {
	Title   string
	Content string
}

const testHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Content}}</h1>
</body>
</html>`

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fs      fstest.MapFS
		opts    []Option[TestData]
		wantErr bool
	}{
		{
			name: "successful initialization with default options",
			fs: fstest.MapFS{
				"templates/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts:    nil,
			wantErr: false,
		},
		{
			name: "successful initialization with custom path",
			fs: fstest.MapFS{
				"custom/template1.html": &fstest.MapFile{
					Data: []byte(testHTMLTemplate),
				},
			},
			opts:    []Option[TestData]{WithTemplatesPath[TestData]("custom")},
			wantErr: false,
		},
		{
			name:    "error with non-existent directory",
			fs:      fstest.MapFS{},
			opts:    []Option[TestData]{WithTemplatesPath[TestData]("non-existent")},
			wantErr: false, // Registry creation succeeds, template loading happens later
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewRegistry[TestData](tt.fs, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestHandler(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/test.html": &fstest.MapFile{
			Data: []byte(testHTMLTemplate),
		},
	}

	reg, err := NewRegistry[TestData](fs)
	require.NoError(t, err)

	tests := []struct {
		name    string
		data    TestData
		wantErr bool
	}{
		{
			name: "successful template execution",
			data: TestData{
				Title:   "Test Title",
				Content: "Test Content",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler, err := reg.Get("test")
			require.NoError(t, err)

			var buf bytes.Buffer
			err = handler.Execute(context.Background(), &buf, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, buf.String(), tt.data.Title)
			assert.Contains(t, buf.String(), tt.data.Content)
		})
	}
}
