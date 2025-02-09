package templator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrTemplateNotFound_Error(t *testing.T) {
	t.Parallel()

	e := ErrTemplateNotFound{Name: "foo"}

	got := e.Error()
	assert.Equal(t, "template 'foo' not found", got)
}

func TestErrTemplateExecution_Error(t *testing.T) {
	t.Parallel()

	e := ErrTemplateExecution{
		Name: "foo",
		Err:  errors.New("bar"),
	}

	got := e.Error()
	assert.Equal(t, "failed to execute template 'foo': 'bar'", got)
}
