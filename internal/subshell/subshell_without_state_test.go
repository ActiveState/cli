// +build !state

package subshell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsActivated(t *testing.T) {
	assert.False(t, IsActivated(), "Test environment is not in an activated state")
}
