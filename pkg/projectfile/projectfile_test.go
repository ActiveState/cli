package projectfile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func setCwd(t *testing.T, subdir string) {
	cwd, err := environment.GetRootPath()
	require.NoError(t, err, "Should fetch cwd")
	path := filepath.Join(cwd, "pkg", "projectfile", "testdata")
	if subdir != "" {
		path = filepath.Join(path, subdir)
	}
	err = os.Chdir(path)
	require.NoError(t, err, "Should change dir without issue.")
}

func TestProjectStruct(t *testing.T) {
	project := Project{}
	dat := strings.TrimSpace(`
name: valueForName
owner: valueForOwner
namespace: valueForNamespace
version: valueForVersion
environments: valueForEnvironments`)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", project.Name, "Name should be set")
	assert.Equal(t, "valueForOwner", project.Owner, "Owner should be set")
	assert.Equal(t, "valueForNamespace", project.Namespace, "Namespace should be set")
	assert.Equal(t, "valueForVersion", project.Version, "Version should be set")
	assert.Equal(t, "valueForEnvironments", project.Environments, "Environments should be set")
	assert.Equal(t, "", project.Path(), "Path should be empty")
}

func TestPlatformStruct(t *testing.T) {
	platform := Platform{}
	dat := strings.TrimSpace(`
name: valueForName
os: valueForOS
version: valueForVersion
architecture: valueForArch`)

	err := yaml.Unmarshal([]byte(dat), &platform)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", platform.Name, "Name should be set")
	assert.Equal(t, "valueForOS", platform.Os, "OS should be set")
	assert.Equal(t, "valueForVersion", platform.Version, "Version should be set")
	assert.Equal(t, "valueForArch", platform.Architecture, "Architecture should be set")
}

func TestBuildStruct(t *testing.T) {
	build := make(Build)
	dat := strings.TrimSpace(`
key1: val1
key2: val2`)

	err := yaml.Unmarshal([]byte(dat), &build)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "val1", build["key1"], "Key1 should be set")
	assert.Equal(t, "val2", build["key2"], "Key2 should be set")
}

func TestLanguageStruct(t *testing.T) {
	language := Language{}
	dat := strings.TrimSpace(`
name: valueForName
version: valueForVersion`)

	err := yaml.Unmarshal([]byte(dat), &language)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", language.Name, "Name should be set")
	assert.Equal(t, "valueForVersion", language.Version, "Version should be set")
}

func TestConstraintStruct(t *testing.T) {
	constraint := Constraint{}
	dat := strings.TrimSpace(`
platform: valueForPlatform
environment: valueForEnvironment`)

	err := yaml.Unmarshal([]byte(dat), &constraint)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForPlatform", constraint.Platform, "Platform should be set")
	assert.Equal(t, "valueForEnvironment", constraint.Environment, "Environment should be set")
}

func TestPackageStruct(t *testing.T) {
	pkg := Package{}
	dat := strings.TrimSpace(`
name: valueForName
version: valueForVersion`)

	err := yaml.Unmarshal([]byte(dat), &pkg)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", pkg.Name, "Name should be set")
	assert.Equal(t, "valueForVersion", pkg.Version, "Version should be set")
}

func TestEventStruct(t *testing.T) {
	event := Event{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForValue`)

	err := yaml.Unmarshal([]byte(dat), &event)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", event.Name, "Name should be set")
	assert.Equal(t, "valueForValue", event.Value, "Value should be set")
}

func TestScriptStruct(t *testing.T) {
	script := Script{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForScript
standalone: true`)

	err := yaml.Unmarshal([]byte(dat), &script)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", script.Name, "Name should be set")
	assert.Equal(t, "valueForScript", script.Value, "Script should be set")
	assert.True(t, script.Standalone, "Standalone should be set")
}

func TestParse(t *testing.T) {
	rootpath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	project, err := Parse(filepath.Join(rootpath, "activestate.yml.nope"))
	assert.NotNil(t, err, "Should throw an error")

	project, err = Parse(filepath.Join(rootpath, "test", "activestate.yaml"))
	assert.Nil(t, err, "Should not throw an error")

	assert.NotEmpty(t, project.Name, "Name should be set")
	assert.NotEmpty(t, project.Owner, "Owner should be set")
	assert.NotEmpty(t, project.Namespace, "Namespace should be set")
	assert.NotEmpty(t, project.Version, "Version should be set")
	assert.NotEmpty(t, project.Platforms, "Platforms should be set")
	assert.NotEmpty(t, project.Environments, "Environments should be set")

	assert.NotEmpty(t, project.Platforms[0].Name, "Platform name should be set")
	assert.NotEmpty(t, project.Platforms[0].Os, "Platform OS name should be set")
	assert.NotEmpty(t, project.Platforms[0].Architecture, "Platform architecture name should be set")
	assert.NotEmpty(t, project.Platforms[0].Libc, "Platform libc name should be set")
	assert.NotEmpty(t, project.Platforms[0].Compiler, "Platform compiler name should be set")

	assert.NotEmpty(t, project.Languages[0].Name, "Language name should be set")
	assert.NotEmpty(t, project.Languages[0].Version, "Language version should be set")

	assert.NotEmpty(t, project.Languages[0].Packages[0].Name, "Package name should be set")
	assert.NotEmpty(t, project.Languages[0].Packages[0].Version, "Package version should be set")

	assert.NotEmpty(t, project.Languages[0].Packages[0].Build, "Package build should be set")
	assert.NotEmpty(t, project.Languages[0].Packages[0].Build["debug"], "Build debug should be set")

	assert.NotEmpty(t, project.Languages[0].Packages[1].Build, "Package build should be set")
	assert.NotEmpty(t, project.Languages[0].Packages[1].Build["override"], "Build override should be set")

	assert.NotEmpty(t, project.Languages[0].Constraints.Platform, "Platform constraint should be set")
	assert.NotEmpty(t, project.Languages[0].Constraints.Environment, "Environment constraint should be set")

	assert.NotEmpty(t, project.Variables[0].Name, "Variable name should be set")
	assert.NotNil(t, project.Variables[0].Value.StaticValue, "Variable value should be set")

	assert.NotEmpty(t, project.Events[0].Name, "Event name should be set")
	assert.NotEmpty(t, project.Events[0].Value, "Event value should be set")

	assert.NotEmpty(t, project.Scripts[0].Name, "Script name should be set")
	assert.NotEmpty(t, project.Scripts[0].Value, "Script value should be set")
	assert.False(t, project.Scripts[0].Standalone, "Standalone value should be set, but false")

	assert.NotEmpty(t, project.Path(), "Path should be set")
}

func TestSave(t *testing.T) {
	rootpath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(rootpath, "test", "activestate.yaml")
	project, failure := Parse(path)
	assert.Nil(t, failure, "unexpected failure parsing our yaml file")

	tmpfile, err := ioutil.TempFile("", "test")
	assert.NoError(t, err, "Should create a temp file")

	project.path = tmpfile.Name()
	project.Save()

	stat, err := tmpfile.Stat()
	assert.NoError(t, err, "Should be able to stat file")

	err = tmpfile.Close()
	assert.NoError(t, err, "Should close our temp file")

	assert.FileExists(t, tmpfile.Name(), "Project file is saved")
	assert.NotZero(t, stat.Size(), "Project file should have data")

	os.Remove(tmpfile.Name())
}

// Call getProjectFilePath
func TestGetProjectFilePath(t *testing.T) {
	Reset()

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	cwd, err := os.Getwd()
	assert.NoError(t, err, "Should fetch cwd")
	os.Chdir(filepath.Join(root, "test"))

	configPath, failure := getProjectFilePath()
	require.Nil(t, failure)
	expectedPath := filepath.Join(root, "test", constants.ConfigFileName)
	assert.Equal(t, expectedPath, configPath, "Project path is properly detected")

	os.Setenv(constants.ProjectEnvVarName, "/some/path")
	defer os.Unsetenv(constants.ProjectEnvVarName)
	configPath, failure = getProjectFilePath()
	require.Nil(t, failure)
	assert.Equal(t, "/some/path", configPath, "Project path is properly detected using the ProjectEnvVarName")

	os.Chdir(cwd) // restore
}

// TestGet the config
func TestGet(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(root, "test"))

	config := Get()
	assert.NotNil(t, config, "Config should be set")
	assert.NotEqual(t, "", os.Getenv(constants.ProjectEnvVarName), "The project env var should be set")

	os.Chdir(cwd) // restore

	Reset()
}

func TestGetActivated(t *testing.T) {
	root, _ := environment.GetRootPath()
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(root, "test"))

	config1 := Get()
	assert.Equal(t, filepath.Join(root, "test", constants.ConfigFileName), os.Getenv(constants.ProjectEnvVarName), "The activated state's config file is set")

	os.Chdir(root)
	config2, fail := GetSafe()
	assert.NoError(t, fail.ToError(), "No error even if no activestate.yaml does not exist")
	assert.Equal(t, config1, config2, "The same activated state is returned")

	expected := filepath.Join(root, "test", constants.ConfigFileName)
	actual := os.Getenv(constants.ProjectEnvVarName)
	assert.Equal(t, expected, actual, "The activated state's config file is still set properly")

	os.Chdir(cwd) // restore

	Reset()
}

func TestParseVersion(t *testing.T) {
	setCwd(t, "")
	version, fail := ParseVersion()
	require.NoError(t, fail.ToError())
	assert.Empty(t, version, "No version exists")

	setCwd(t, "withversion")
	version, fail = ParseVersion()
	require.NoError(t, fail.ToError())
	assert.NotEmpty(t, version, "Version exists")

	setCwd(t, "withbadversion")
	version, fail = ParseVersion()
	assert.Error(t, fail.ToError())
	assert.Equal(t, FailInvalidVersion.Name, fail.Type.Name, "Fails with FailInvalidVersion")

	path, err := ioutil.TempDir("", "ParseVersionTest")
	require.NoError(t, err)
	os.Chdir(path)
	version, fail = ParseVersion()
	require.NoError(t, fail.ToError())
	assert.Empty(t, version, "No version exists, because no project file exists")
}
