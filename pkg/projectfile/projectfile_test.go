package projectfile

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
)

func setCwd(t *testing.T, subdir string) {
	err := os.Chdir(getWd(t, subdir))
	require.NoError(t, err, "Should change dir without issue.")
}

func getWd(t *testing.T, subdir string) string {
	cwd, err := environment.GetRootPath()
	require.NoError(t, err, "Should fetch cwd")
	path := filepath.Join(cwd, "pkg", "projectfile", "testdata")
	if subdir != "" {
		path = filepath.Join(path, subdir)
	}
	return path
}

func TestProjectStruct(t *testing.T) {
	project := Project{}
	dat := strings.TrimSpace(`
project: valueForProject
namespace: valueForNamespace
version: valueForVersion
environments: valueForEnvironments`)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForProject", project.Project, "Project should be set")
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
os: valueForOS
platform: valueForPlatform
environment: valueForEnvironment`)

	err := yaml.Unmarshal([]byte(dat), &constraint)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForOS", constraint.OS, "Os should be set")
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
language: bash
value: valueForScript
standalone: true`)

	err := yaml.Unmarshal([]byte(dat), &script)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", script.Name, "Name should be set")
	assert.Equal(t, language.Bash, script.Language, "Language should match")
	assert.Equal(t, "valueForScript", script.Value, "Script should be set")
	assert.True(t, script.Standalone, "Standalone should be set")
}

func TestConstantStruct(t *testing.T) {
	constant := Constant{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForConstant`)

	err := yaml.Unmarshal([]byte(dat), &constant)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", constant.Name, "Name should be set")
	assert.Equal(t, "valueForConstant", constant.Value, "Constant should be set")
}

func TestSecretStruct(t *testing.T) {
	secret := Secret{}
	dat := strings.TrimSpace(`
name: valueForName
description: valueForDescription`)

	err := yaml.Unmarshal([]byte(dat), &secret)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", secret.Name, "Name should be set")
	assert.Equal(t, "valueForDescription", secret.Description, "Description should be set")
}

func TestParse(t *testing.T) {
	rootpath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	project, err := Parse(filepath.Join(rootpath, "activestate.yml.nope"))
	assert.NotNil(t, err, "Should throw an error")

	project, err = Parse(filepath.Join(rootpath, "pkg", "projectfile", "testdata", "activestate.yaml"))
	assert.Nil(t, err, "Should not throw an error")

	assert.NotEmpty(t, project.Project, "Project should be set")
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

	assert.NotEmpty(t, project.Languages[0].Constraints.OS, "Platform constraint should be set")
	assert.NotEmpty(t, project.Languages[0].Constraints.Platform, "Platform constraint should be set")
	assert.NotEmpty(t, project.Languages[0].Constraints.Environment, "Environment constraint should be set")

	assert.NotEmpty(t, project.Constants[0].Name, "Constant name should be set")
	assert.NotEmpty(t, project.Constants[0].Value, "Constant value should be set")

	assert.NotEmpty(t, project.Secrets.User[0].Name, "Variable name should be set")
	assert.NotEmpty(t, project.Secrets.Project[0].Name, "Variable name should be set")

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

	path := filepath.Join(rootpath, "pkg", "projectfile", "testdata", "activestate.yaml")
	project, failure := Parse(path)
	assert.Nil(t, failure, "unexpected failure parsing our yaml file")

	tmpfile, err := ioutil.TempFile("", "test")
	assert.NoError(t, err, "Should create a temp file")

	project.path = tmpfile.Name()
	project.Save()

	stat, err := tmpfile.Stat()
	assert.NoError(t, err, "Should be able to stat file")

	projectURL := project.Project
	project.Project = "thisisnotatallaprojectURL"
	fail := project.Save()
	assert.Error(t, fail.ToError(), "Saving project should fail due to bad projectURL format")
	project.Project = projectURL
	fail = project.Save()
	assert.NoError(t, fail.ToError(), "Saving project should now pass")

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
	defer os.Chdir(cwd) // restore
	os.Chdir(filepath.Join(root, "pkg", "projectfile", "testdata"))

	configPath, fail := GetProjectFilePath()
	require.Nil(t, fail)
	expectedPath := filepath.Join(root, "pkg", "projectfile", "testdata", constants.ConfigFileName)
	assert.Equal(t, expectedPath, configPath, "Project path is properly detected")

	defer os.Unsetenv(constants.ProjectEnvVarName)

	os.Setenv(constants.ProjectEnvVarName, "/some/path")
	configPath, fail = GetProjectFilePath()
	require.NotNil(t, fail)
	require.Equal(t, FailNoProjectFromEnv.Name, fail.Type.Name, "Failure types should match")

	expectedPath = filepath.Join(root, "pkg", "projectfile", "testdata", constants.ConfigFileName)
	os.Setenv(constants.ProjectEnvVarName, expectedPath)
	configPath, fail = GetProjectFilePath()
	require.Nil(t, fail)
	assert.Equal(t, expectedPath, configPath, "Project path is properly detected using the ProjectEnvVarName")

	os.Unsetenv(constants.ProjectEnvVarName)
	tmpDir, err := ioutil.TempDir("", "")
	assert.NoError(t, err, "Should create temp dir")
	defer os.RemoveAll(tmpDir)
	os.Chdir(tmpDir)
	_, fail = GetProjectFilePath()
	assert.Error(t, fail.ToError(), "GetProjectFilePath should fail")
	viper.SetDefault("default_project_path", expectedPath)
	configPath, fail = GetProjectFilePath()
	assert.NoError(t, fail.ToError(), "GetProjectFilePath should succeed")
	assert.Equal(t, expectedPath, configPath, "Project path is properly detected using default path from config")
}

// TestGet the config
func TestGet(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(root, "pkg", "projectfile", "testdata"))

	config := Get()
	assert.NotNil(t, config, "Config should be set")
	assert.NotEqual(t, "", os.Getenv(constants.ProjectEnvVarName), "The project env var should be set")

	os.Chdir(cwd) // restore

	Reset()
}

func TestGetActivated(t *testing.T) {
	root, _ := environment.GetRootPath()
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(root, "pkg", "projectfile", "testdata"))

	config1 := Get()
	assert.Equal(t, filepath.Join(root, "pkg", "projectfile", "testdata", constants.ConfigFileName), os.Getenv(constants.ProjectEnvVarName), "The activated state's config file is set")

	os.Chdir(root)
	config2, fail := GetSafe()
	assert.NoError(t, fail.ToError(), "No error even if no activestate.yaml does not exist")
	assert.Equal(t, config1, config2, "The same activated state is returned")

	expected := filepath.Join(root, "pkg", "projectfile", "testdata", constants.ConfigFileName)
	actual := os.Getenv(constants.ProjectEnvVarName)
	assert.Equal(t, expected, actual, "The activated state's config file is still set properly")

	os.Chdir(cwd) // restore

	Reset()
}

func TestParseVersionInfo(t *testing.T) {
	versionInfo, fail := ParseVersionInfo(filepath.Join(getWd(t, ""), constants.ConfigFileName))
	require.NoError(t, fail.ToError())
	assert.Nil(t, versionInfo, "No version exists")

	versionInfo, fail = ParseVersionInfo(filepath.Join(getWd(t, "withversion"), constants.ConfigFileName))
	require.NoError(t, fail.ToError())
	assert.NotNil(t, versionInfo, "Version exists")

	versionInfo, fail = ParseVersionInfo(filepath.Join(getWd(t, "withbadversion"), constants.ConfigFileName))
	assert.Error(t, fail.ToError())
	assert.Equal(t, FailInvalidVersion.Name, fail.Type.Name, "Fails with FailInvalidVersion")

	path, err := ioutil.TempDir("", "ParseVersionInfoTest")
	require.NoError(t, err)
	versionInfo, fail = ParseVersionInfo(filepath.Join(path, constants.ConfigFileName))
	require.NoError(t, fail.ToError())
	assert.Nil(t, versionInfo, "No version exists, because no project file exists")
}

func TestSetCommitInYAML(t *testing.T) {
	exampleYAML := []byte(`
junk: xgarbage
project: https://example.com/xowner/xproject?commitID=123
123: xvalue
`)
	expectedYAML := bytes.Replace(exampleYAML, []byte("123"), []byte("987"), 1) // must be 1

	_, fail := setCommitInYAML(exampleYAML, "")
	assert.Equal(t, failures.FailDeveloper.Name, fail.Type.Name)

	_, fail = setCommitInYAML([]byte(""), "123")
	assert.Equal(t, FailSetCommitID.Name, fail.Type.Name)

	out0, fail := setCommitInYAML(exampleYAML, "987")
	assert.NoError(t, fail.ToError())
	assert.Equal(t, string(expectedYAML), string(out0))

	exampleYAMLNoID := bytes.Replace(exampleYAML, []byte("?commitID=123"), nil, 1)
	out1, fail := setCommitInYAML(exampleYAMLNoID, "987")
	assert.NoError(t, fail.ToError())
	assert.Equal(t, string(expectedYAML), string(out1))
}

func TestSetCommitInYAML_NoCommitID(t *testing.T) {
	exampleYAML := []byte(`
junk: xgarbage
project: https://example.com/xowner/xproject
123: xvalue
`)
	expectedYAML := []byte(`
junk: xgarbage
project: https://example.com/xowner/xproject?commitID=123
123: xvalue
`)

	out, fail := setCommitInYAML(exampleYAML, "123")
	assert.NoError(t, fail.ToError())
	assert.Equal(t, string(expectedYAML), string(out))
}

func TestNewProjectfile(t *testing.T) {
	dir, err := ioutil.TempDir("", "projectfile-test")
	assert.NoError(t, err, "Should be no error when getting a temp directory")
	os.Chdir(dir)

	pjFile, fail := CreateWithProjectURL("https://platform.activestate.com/xowner/xproject", dir)
	assert.NoError(t, fail.ToError(), "There should be no error when loading from a path")
	assert.Equal(t, "helloWorld", pjFile.Scripts[0].Name)

	_, fail = CreateWithProjectURL("https://platform.activestate.com/xowner/xproject", "")
	assert.Error(t, fail.ToError(), "We don't accept blank paths")

	setCwd(t, "")
	dir, err = os.Getwd()
	assert.NoError(t, err, "Should be no error when getting the CWD")
	_, fail = CreateWithProjectURL("https://platform.activestate.com/xowner/xproject", dir)
	assert.Error(t, fail.ToError(), "Cannot create new project if existing as.yaml ...exists")
}

func TestValidateProjectURL(t *testing.T) {
	fail := ValidateProjectURL("https://example.com/xowner/xproject")
	assert.Error(t, fail.ToError(), "This should be an invalid project URL")

	fail = ValidateProjectURL("https://platform.activestate.com/xowner/xproject")
	assert.Nil(t, fail, "This should not be an invalid project URL")

	fail = ValidateProjectURL("https://platform.activestate.com/commit/commitid")
	assert.Nil(t, fail, "This should not be an invalid project URL using the commit path")
}
