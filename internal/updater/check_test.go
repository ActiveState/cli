package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckForAndApplyUpdates(t *testing.T) {
	CheckForAndApplyUpdates()
	assert.True(t, true, "no panic")
}
