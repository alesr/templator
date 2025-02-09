package templator

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator_Error(t *testing.T) {
	t.Parallel()

	validatorError := ValidationError{
		TemplateName: "dummy-template-name",
		FieldPath:    "dummy-file-path",
		Err:          errors.New("dummy-error"),
	}

	got := validatorError.Error()

	assert.Equal(t, "template 'dummy-template-name' validation error: 'dummy-file-path' - 'dummy-error'", got)
}

func TestUniqueFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		givenFields []string
		expect      []string
	}{
		{
			name:        "unique fields",
			givenFields: []string{"foo", "bar"},
			expect:      []string{"foo", "bar"},
		},
		{
			name:        "duplicated fields",
			givenFields: []string{"foo", "bar", "foo"},
			expect:      []string{"foo", "bar"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := uniqueFields(tc.givenFields)

			assert.ElementsMatch(t, tc.expect, got)
		})
	}
}

func Test_validateField(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		testField string
	}

	testCases := []struct {
		name      string
		typ       reflect.Type
		fieldPath string
		wantErr   bool
	}{
		{
			name:      "nil type",
			typ:       nil,
			fieldPath: "testField",
			wantErr:   true,
		},
		{
			name:      "non-struct type",
			typ:       reflect.TypeOf(""),
			fieldPath: "testField",
			wantErr:   true,
		},
		{
			name:      "field is a struct",
			typ:       reflect.TypeOf(testStruct{}),
			fieldPath: "testField",
			wantErr:   false,
		},
		{
			name:      "field is a pointer",
			typ:       reflect.TypeOf(&testStruct{}),
			fieldPath: "testField",
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateField(tc.typ, tc.fieldPath)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
