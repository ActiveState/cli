package virtualenvironment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	rtmock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
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

func TestEnv(t *testing.T) {
	setup(t)
	defer teardown()

	os.Setenv(constants.DisableRuntime, "true")
	defer os.Unsetenv(constants.DisableRuntime)

	os.Setenv(constants.ProjectEnvVarName, projectfile.Get().Path())
	defer os.Unsetenv(constants.ProjectEnvVarName)

	venv := New(runtime.DisabledRuntime)
	env, err := venv.GetEnv(false, true, filepath.Dir(projectfile.Get().Path()))
	require.NoError(t, err)

	assert.NotContains(t, env, constants.ProjectEnvVarName)
	assert.NotEmpty(t, env[constants.ActivatedStateIDEnvVarName])
	assert.NotEmpty(t, venv.ActivationID())
}

func TestInheritEnv_MultipleEquals(t *testing.T) {
	key := "MULTIPLEEQUALS"
	value := "one=two two=three three=four"

	os.Setenv(key, value)
	defer os.Unsetenv(key)

	env := map[string]string{}
	updated := inheritEnv(env)

	assert.Equal(t, value, updated[key])
}
