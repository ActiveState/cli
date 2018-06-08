package remove

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

// This is mostly a clone of the state/hooks/remove/remove_test.go file. Any
// tests added, modified, or removed in that file should be applied here and
// vice-versa.

// Copies the activestate config file in the root test/ directory into the local
// config directory, reads the config file as a project, and returns that
// project.
func getTestProject(t *testing.T) *projectfile.Project {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Got root path")
	src := filepath.Join(root, "test", constants.ConfigFileName)
	dst := filepath.Join(config.GetDataDir(), constants.ConfigFileName)
	fail := fileutils.CopyFile(src, dst)
	assert.Nil(t, fail, "Copied test activestate config file")
	project, err := projectfile.Parse(dst)
	assert.NoError(t, err, "Parsed test config file")
	return project
}

func setup(t *testing.T) {
	Args.Identifier = ""
	testPromptResultOverride = ""
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
	projectfile.Reset()
}

func TestExecute(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	assert := assert.New(t)
	Command.Execute()
	assert.Equal(true, true, "Execute didn't panic")
}

func TestRemoveByHashCmd(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	varName := "foo"

	variable := projectfile.Variable{Name: varName, Value: "value"}
	project.Variables = append(project.Variables, variable)
	project.Save()

	hash, _ := variable.Hash()
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{hash})
	Command.Execute()
	Cc.SetArgs([]string{})

	project = projectfile.Get()
	mappedVariables, _ := variables.HashVariablesFiltered(project.Variables, []string{varName})
	assert.Equal(t, 0, len(mappedVariables), fmt.Sprintf("No variables should be found of name: '%v'", varName))
}

func TestRemoveByNameCmd(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	varName := "foo"

	variable := projectfile.Variable{Name: varName, Value: "value"}
	project.Variables = append(project.Variables, variable)
	project.Save()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{varName})
	Command.Execute()
	Cc.SetArgs([]string{})

	project = projectfile.Get()
	mappedVariables, _ := variables.HashVariablesFiltered(project.Variables, []string{varName})
	assert.Equal(t, 0, len(mappedVariables), fmt.Sprintf("No variables should be found of name: '%v', found: %v", varName, mappedVariables))
}

func TestRemovePrompt(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	options, optionsMap, err := variables.PromptOptions("DEBUG")
	print.Formatted("\nmap1: %v\n", optionsMap)
	assert.NoError(t, err, "Should be able to get prompt options")

	testPromptResultOverride = options[0]

	removed := removeByPrompt("DEBUG")
	assert.NotNil(t, removed, "Received a removed variable")

	hash, _ := removed.Hash()
	assert.Equal(t, optionsMap[testPromptResultOverride], hash, "Should have removed one variable")
}

func TestRemoveByHash(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	variableLen := len(project.Variables)

	hash, err := project.Variables[0].Hash()
	assert.NoError(t, err, "Should get hash")
	removed := removeByHash(hash)
	assert.NotNil(t, removed, "Received a removed variable")

	project = projectfile.Get()
	assert.Equal(t, variableLen-1, len(project.Variables), "One variable should have been removed")
}

func TestRemovebyName(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	variableLen := len(project.Variables)

	removed := removeByName(project.Variables[0].Name)
	assert.NotNil(t, removed, "Received a removed variable")

	assert.Equal(t, variableLen-1, len(project.Variables), "One variable should have been removed")
}

// This test shoudln't remove anything as there are multiple variables defined for the same variable name
func TestRemoveByNameFailCmd(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	varName := "foo"

	variable1 := projectfile.Variable{Name: varName, Value: "bar"}
	variable2 := projectfile.Variable{Name: varName, Value: "baz", Constraints: projectfile.Constraint{Platform: "windows"}}
	project.Variables = append(project.Variables, variable1)
	project.Variables = append(project.Variables, variable2)
	project.Save()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{varName})
	Command.Execute()
	Cc.SetArgs([]string{})

	mappedVariables, _ := variables.HashVariablesFiltered(project.Variables, []string{varName})
	assert.Equal(t, 2, len(mappedVariables), fmt.Sprintf("There should still be two variables of the same name in the config: '%v'", varName))
}

func TestRemoveNonExistant(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	_, _, err := variables.PromptOptions("DEBUG")
	assert.NoError(t, err, "Should be able to get prompt options")

	testPromptResultOverride = "does-not-exist"

	removed := removeByPrompt("DEBUG")
	assert.Nil(t, removed, "Could not remove non-existant variable")
}
