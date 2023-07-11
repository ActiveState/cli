package ptr

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTo(t *testing.T) {
	stringV := "stringV"
	stringP := To(stringV)
	assert.Equal(t, "stringV", *stringP)
	assert.True(t, reflect.ValueOf(stringP).Kind() == reflect.Ptr, "Expected result to be a pointer")

	intV := 999
	intP := To(intV)
	assert.Equal(t, 999, *intP)
	assert.True(t, reflect.ValueOf(intP).Kind() == reflect.Ptr, "Expected result to be a pointer")
}

func TestFrom(t *testing.T) {
	stringP := To("stringP")
	assert.Equal(t, "stringP", From(stringP, ""))

	var stringPNil *string
	assert.Equal(t, "fallback", From(stringPNil, "fallback"))

	intP := To(999)
	assert.Equal(t, 999, From(intP, -1))

	var intPNil *int
	assert.Equal(t, -1, From(intPNil, -1))
}

func TestClone(t *testing.T) {
	stringP := To("stringP")
	cloneStringP := Clone(stringP)
	assert.Equal(t, "stringP", *cloneStringP)
	assert.NotSame(t, stringP, cloneStringP)

	var stringPNil *string
	cloneStringPNil := Clone(stringPNil)
	assert.Nil(t, cloneStringPNil)

	intP := To(999)
	cloneIntP := Clone(intP)
	assert.Equal(t, 999, *cloneIntP)
	assert.NotSame(t, intP, cloneIntP)
}
