package activate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/stretchr/testify/assert"
)

func init() {
	if os.Getenv("CI") == "true" {
		os.Setenv("SHELL", "/bin/bash")
	}
}

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	cwd, _ := os.Getwd() // store
	err := os.Chdir(filepath.Join(environment.GetRootPathUnsafe(), "state", "activate", "testdata"))
	assert.Nil(err, "unable to chdir to testdata dir")

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
	assert.NoError(failures.Handled(), "No failure occurred")

	err = os.Chdir(cwd)
	assert.Nil(err, "Changed back to original cwd")
}
