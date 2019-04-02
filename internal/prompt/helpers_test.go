package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRequired(t *testing.T) {
	assert.Error(t, ValidateRequired(""), "Throws error because value is empty")
	assert.NoError(t, ValidateRequired("foo"), "Doesn't throw an error cause value 'foo' is not empty")
	assert.NoError(t, ValidateRequired(0), "Doesn't throw an error cause value is '0' not empty")
	assert.NoError(t, ValidateRequired(false), "Doesn't throw an error cause value 'false' is not empty")
}

func TestFormatMessageByCols(t *testing.T) {
	assert.Equal(t, "aa\naa\naa", formatMessageByCols("aaaaaa", 3), "Adds linebreaks at col limit")
	assert.Equal(t, "a\naa\naa\na", formatMessageByCols("a\naaaaa", 3), "Adds linebreaks at col limit")
}
