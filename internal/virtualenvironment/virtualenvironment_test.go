package virtualenvironment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
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

	venv := New(nil)
	env, err := venv.GetEnv(false, filepath.Dir(projectfile.Get().Path()))
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

func TestSkipActivateRuntimeEnvironment(t *testing.T) {
	setup(t)
	defer teardown()

	os.Setenv(constants.DisableRuntime, "true")
	defer os.Unsetenv(constants.DisableRuntime)

	project := projectfile.Project{}
	dat := strings.TrimSpace(`
project: "https://platform.activestate.com/string/string?commitID=00010001-0001-0001-0001-000100010001"
languages:
    - name: Python3`)
	yaml.Unmarshal([]byte(dat), &project)
	project.Persist()

	venv := New(nil)
	fail := venv.Activate()
	require.NoError(t, fail.ToError(), "Should activate")
}
