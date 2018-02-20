package projectfile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	C "github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestProjectStruct(t *testing.T) {
	project := Project{}
	dat := strings.TrimSpace(`
name: valueForName
owner: valueForOwner
version: valueForVersion
environments: valueForEnvironments`)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", project.Name, "Name should be set")
	assert.Equal(t, "valueForOwner", project.Owner, "Owner should be set")
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

func TestVariableStruct(t *testing.T) {
	variable := Variable{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForValue`)

	err := yaml.Unmarshal([]byte(dat), &variable)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", variable.Name, "Name should be set")
	assert.Equal(t, "valueForValue", variable.Value, "Value should be set")
}

func TestHookStruct(t *testing.T) {
	hook := Hook{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForValue`)

	err := yaml.Unmarshal([]byte(dat), &hook)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", hook.Name, "Name should be set")
	assert.Equal(t, "valueForValue", hook.Value, "Value should be set")
}

func TestCommandStruct(t *testing.T) {
	command := Command{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForCommand`)

	err := yaml.Unmarshal([]byte(dat), &command)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", command.Name, "Name should be set")
	assert.Equal(t, "valueForCommand", command.Value, "Command should be set")
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
	assert.NotEmpty(t, project.Variables[0].Value, "Variable value should be set")

	assert.NotEmpty(t, project.Hooks[0].Name, "Hook name should be set")
	assert.NotEmpty(t, project.Hooks[0].Value, "Hook value should be set")

	assert.NotEmpty(t, project.Commands[0].Name, "Command name should be set")
	assert.NotEmpty(t, project.Commands[0].Value, "Command value should be set")

	assert.NotEmpty(t, project.Path(), "Path should be set")
}

func TestSave(t *testing.T) {
	rootpath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(rootpath, "test", "activestate.yaml")
	project, err := Parse(path)
	assert.NoError(t, err, "Should parse our yaml file")

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

// Call GetProjectFilePath but doesn't exist
func TestGetFail(t *testing.T) {
	configFilename = "activestate.yml.does_not_exist"
	config, _ := Get()
	assert.Nil(t, config, "Config should not be set.")
	configFilename = C.ConfigFileName // reset
}

// TestGet the config
func TestGet(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	config, _ := Get()
	hash := projectHash
	assert.NotNil(t, config, "Config should be set")
	assert.NotEqual(t, hash, "", "Cache hash should be set")
}

//Test cache reset
func TestGetCache(t *testing.T) {
	configFilename = "activestate.yml.sample"
	Get()
	originalhash := projectHash
	Get()
	newHash := projectHash
	assert.Equal(t, originalhash, newHash, "Both hashes should not change")
	configFilename = C.ConfigFileName // reset
}

//Test cache reset
func TestGetNewCache(t *testing.T) {
	config, _ := Get()
	originalhash := projectHash
	config.Languages[0].Version = "0.0.0"
	configFilename = "activestate.yml.sample.delete"
	testConfigFile := GetProjectFilePath()
	config.path = testConfigFile
	config.Save()
	Get()
	newHash := projectHash
	os.Remove(testConfigFile)
	assert.NotEqual(t, originalhash, newHash, "Hashes should be different")
	configFilename = C.ConfigFileName // reset
}

//Test cache reset
func TestGetCacheReset(t *testing.T) {
	currentProject = nil // reset
	configFilename = "activestate.yml.doesnotexist"
	config, _ := Get()
	deletedHash := projectHash
	assert.Nil(t, config, "Config should NOT be set")
	assert.Equal(t, deletedHash, "", "Hash should be empty")
	configFilename = C.ConfigFileName // reset
}

func TestGetActivated(t *testing.T) {
	root, _ := environment.GetRootPath()
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(root, "test"))

	config1, _ := Get()
	os.Chdir(root)
	config2, err := Get()
	assert.NoError(t, err, "No error even if no activestate.yaml does not exist")
	assert.Equal(t, config1, config2, "The same activated state is returned if activestate.yaml does not exist")

	os.Chdir(cwd) // restore
}

// Call GetProjectFilePath
func TestGetProjectFilePath(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	configPath := GetProjectFilePath()
	expectedPath := filepath.Join(root, "test", C.ConfigFileName)
	assert.Equal(t, expectedPath, configPath, "Project path is properly detected")
}
