package templator

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedMethods(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"templates/home.html": &fstest.MapFile{
			Data: []byte(testHTMLTemplate),
		},
		"templates/components/menu.html": &fstest.MapFile{
			Data: []byte("<nav>{{.Title}}</nav>"),
		},
	}

	reg, err := NewRegistry[TestData](fs)
	require.NoError(t, err)

	tests := []struct {
		name    string
		method  func() (*Handler[TestData], error)
		wantErr bool
	}{
		{
			name:    "GetHome",
			method:  reg.GetHome,
			wantErr: false,
		},
		{
			name:    "GetComponentsMenu",
			method:  reg.GetComponentsMenu,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler, err := tt.method()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, handler)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, handler)
		})
	}
}
