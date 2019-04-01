package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatMessageByCols(t *testing.T) {
	assert.Equal(t, "aa\naa\naa", formatMessageByCols("aaaaaa", 3), "Adds linebreaks at col limit")
	assert.Equal(t, "a\naa\naa\na", formatMessageByCols("a\naaaaa", 3), "Adds linebreaks at col limit")
}
