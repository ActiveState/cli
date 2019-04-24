// +build !windows

package ct

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnsiText(t *testing.T) {
	assert.Equal(t, "", ansiText(None, false, None, false))
	assert.Equal(t, "\x1b[0;31m", ansiText(Red, false, None, false))
	assert.Equal(t, "\x1b[0;31;1m", ansiText(Red, true, None, false))
	assert.Equal(t, "\x1b[0;42m", ansiText(None, false, Green, false))
	assert.Equal(t, "\x1b[0;31;42m", ansiText(Red, false, Green, false))
	assert.Equal(t, "\x1b[0;31;1;42m", ansiText(Red, true, Green, false))
}
