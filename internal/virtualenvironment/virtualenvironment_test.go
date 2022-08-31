package virtualenvironment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	rtmock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

var rtMock *rtmock.Mock

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")

	err = os.Chdir(filepath.Join(root, "internal", "virtualenvironment", "testdata"))
	assert.NoError(t, err, "unable to chdir to testdata dir")

	rtMock = rtmock.Init()
	rtMock.MockFullRuntime()

	os.Unsetenv(constants.ActivatedStateEnvVarName)
	os.Unsetenv(constants.ActivatedStateIDEnvVarName)
}

func teardown() {
	projectfile.Reset()
	rtMock.Close()
}
