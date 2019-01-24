// +build !state

package projectfile

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
)

// Call getProjectFilePath but doesn't exist
func TestGetFail(t *testing.T) {
	config, _ := GetSafe()
	assert.Nil(t, config, "Config should not be set.")
	assert.Equal(t, "", os.Getenv(constants.ProjectEnvVarName), "The state should not be activated")

	Reset()
}
