package virtualenvironment

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/shimming"
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
}

func teardown() {
	projectfile.Reset()
	rtMock.Close()
}

func TestPersist(t *testing.T) {
	setup(t)
	defer teardown()

	v1 := Get()
	v2 := Get()
	assert.True(t, v1 == v2, "Should return same pointer")
}

func TestEvents(t *testing.T) {
	venv := Init()
	onDownloadCalled := false
	onInstallCalled := false

	venv.OnDownloadArtifacts(func() { onDownloadCalled = true })
	venv.OnInstallArtifacts(func() { onInstallCalled = true })

	venv.onDownloadArtifacts()
	venv.onInstallArtifacts()

	assert.True(t, onDownloadCalled, "OnDownloadArtifacts is triggered")
	assert.True(t, onInstallCalled, "OnInstallArtifacts is triggered")
}

func TestActivate(t *testing.T) {
	setup(t)
	defer teardown()

	venv := Init()
	fail := venv.Activate()
	if runtime.GOOS == "windows" {
		// Since creating symlinks on Windows requires admin privilages for now,
		// test activation should fail.
		require.Error(t, fail, "Symlinking requires admin privilages for now")
	} else {
		require.NoError(t, fail.ToError(), "Should activate")
	}

	setup(t)
	project := &projectfile.Project{
		Name:  "string",
		Owner: "string",
	}
	project.Persist()

	venv = Init()
	fail = venv.Activate()
	if runtime.GOOS == "windows" {
		// Since creating symlinks on Windows requires admin privilages for now,
		// test activation should fail.
		require.Error(t, fail, "Symlinking requires admin privilages for now")
	} else {
		require.NoError(t, fail.ToError(), "Should activate, even if no languages are defined")
	}
}

func TestActivateFailureUnknownLanguage(t *testing.T) {
	setup(t)
	defer teardown()

	project := projectfile.Get()
	project.Languages = append(project.Languages, projectfile.Language{
		Name: "foo",
	})
	project.Persist()

	venv := Init()
	err := venv.Activate()
	assert.Error(t, err, "Should not activate due to unknown language")
}

func TestActivateFailureAlreadyActive(t *testing.T) {
	setup(t)
	defer teardown()

	os.Setenv(constants.ActivatedStateEnvVarName, "test")

	venv := Init()
	failure := venv.Activate()
	namespace := venv.project.Owner() + "/" + venv.project.Name()
	require.NotNil(t, failure, "expected a failure")
	assert.Equal(t, FailAlreadyActive, failure.Type)
	assert.Equal(t, locale.Tr("err_already_active", namespace), failure.Error())
}

func TestActivateShims(t *testing.T) {
	setup(t)
	defer teardown()

	venv := Init()
	env := venv.envPath

	collection := shimming.NewCollection()
	collection.RegisterShim(shimming.NewShim([]string{"binary1"}))

	dir, err := os.Getwd()
	require.NoError(t, err)

	fail := venv.activateShims([]string{dir}, collection)
	require.NoError(t, fail.ToError())

	assert.NotEqual(t, env, venv.envPath, "envPath should have changed")
}

func TestActivateShimsNoResult(t *testing.T) {
	setup(t)
	defer teardown()

	venv := Init()
	env := venv.envPath

	fail := venv.activateShims([]string{}, shimming.NewCollection())
	require.NoError(t, fail.ToError())

	assert.Equal(t, env, venv.envPath, "envPath should not have changed")
}

func TestEnv(t *testing.T) {
	setup(t)
	defer teardown()

	os.Setenv(constants.ProjectEnvVarName, projectfile.Get().Path())

	venv := Init()
	env := venv.GetEnv()

	assert.NotContains(t, env, constants.ProjectEnvVarName)
}
