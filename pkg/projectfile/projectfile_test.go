package projectfile

import (
	"path/filepath"
	"strings"
	"testing"

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
platforms: valueForPlatforms
environments: valueForEnvironments`)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.Nil(t, err, "Should not throw an error")

	assert.Equal(t, "valueForName", project.Name, "Name should be set")
	assert.Equal(t, "valueForOwner", project.Owner, "Owner should be set")
	assert.Equal(t, "valueForVersion", project.Version, "Version should be set")
	assert.Equal(t, "valueForPlatforms", project.Platforms, "Platforms should be set")
	assert.Equal(t, "valueForEnvironments", project.Environments, "Environments should be set")
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

	project, err := Parse(filepath.Join(rootpath, "activestate.yml.sample"))

	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, project.Name, "Name should be set")
	assert.NotEmpty(t, project.Owner, "Owner should be set")
	assert.NotEmpty(t, project.Version, "Version should be set")
	assert.NotEmpty(t, project.Platforms, "Platforms should be set")
	assert.NotEmpty(t, project.Environments, "Environments should be set")

	assert.NotEmpty(t, project.Languages[0].Name, "Language name should be set")
	assert.NotEmpty(t, project.Languages[0].Version, "Language version should be set")

	assert.NotEmpty(t, project.Languages[0].Packages[0].Name, "Package name should be set")
	assert.NotEmpty(t, project.Languages[0].Packages[0].Version, "Package version should be set")

	assert.NotEmpty(t, project.Languages[0].Constraints.Platform, "Platform constraint should be set")
	assert.NotEmpty(t, project.Languages[0].Constraints.Environment, "Environment constraint should be set")

	assert.NotEmpty(t, project.Variables[0].Name, "Variable name should be set")
	assert.NotEmpty(t, project.Variables[0].Value, "Variable value should be set")

	assert.NotEmpty(t, project.Hooks[0].Name, "Hook name should be set")
	assert.NotEmpty(t, project.Hooks[0].Value, "Hook value should be set")

	assert.NotEmpty(t, project.Commands[0].Name, "Command name should be set")
	assert.NotEmpty(t, project.Commands[0].Value, "Command value should be set")
}
