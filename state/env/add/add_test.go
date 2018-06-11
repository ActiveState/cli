package add

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/cmdlets/variables"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

// This is mostly a clone of the state/hooks/add/add_test.go file. Any tests
// added, modified, or removed in that file should be applied here and
// vice-versa.

// Copies the activestate config file in the root test/ directory into the local
// config directory, reads the config file as a project, and returns that
// project.
func getTestProject(t *testing.T) *projectfile.Project {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Got root path")
	src := filepath.Join(root, "test", constants.ConfigFileName)
	dst := filepath.Join(root, "state", "env", "add", "testdata", "generated", "config", constants.ConfigFileName)
	fail := fileutils.CopyFile(src, dst)
	assert.Nil(t, fail, "Copied test activestate config file")
	project, err := projectfile.Parse(dst)
	assert.NoError(t, err, "Parsed test config file")
	return project
}

func TestAddVariablePass(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	project := getTestProject(t)
	project.Persist()

	newVariableName := "foo"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{newVariableName, "bar"})
	Cc.Execute()

	var found = false
	for _, variable := range project.Variables {
		if variable.Name == newVariableName {
			found = true
		}
	}
	assert.True(t, found, fmt.Sprintf("Should find a variable named %v", newVariableName))
}

func TestAddVariableFail(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	project := getTestProject(t)
	project.Persist()

	Cc := Command.GetCobraCmd()
	newVariableName := "foo?!"
	Cc.SetArgs([]string{newVariableName})
	Cc.Execute()

	var found = false
	for _, variable := range project.Variables {
		if variable.Name == newVariableName {
			found = true
		}
	}
	assert.False(t, found, fmt.Sprintf("Should NOT find a variable named %v", newVariableName))
}

// Test it doesn't explode when run with no args
func TestExecute(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	project := getTestProject(t)
	project.Persist()

	Command.Execute()

	assert.Equal(t, true, true, "Execute didn't panic")
}

//
func TestAddVariableFailIdentical(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	project := getTestProject(t)
	project.Persist()

	variableName := "DEBUG"
	value := "true"
	variable1 := projectfile.Variable{Name: variableName, Value: value}
	project.Variables = append(project.Variables, variable1)
	project.Save()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{variableName, value})
	Cc.Execute()

	filteredMappedVariables, _ := variables.HashVariablesFiltered(project.Variables, []string{variableName})

	assert.Equal(t, 1,
		len(filteredMappedVariables),
		fmt.Sprintf("There should be only one variable defined for variablename'%v'", variableName))
}

func TestAddVariableInheritValue(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	project := getTestProject(t)
	project.Persist()
	os.Setenv("foo", "baz")

	newVariableName := "foo"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{newVariableName})
	Cc.Execute()

	var found *projectfile.Variable
	for _, variable := range project.Variables {
		if variable.Name == newVariableName {
			found = &variable
		}
	}
	assert.NotNil(t, found, fmt.Sprintf("Should find a variable named %v", newVariableName))
	assert.Equal(t, os.Getenv("foo"), found.Value, "Variable value should be inherited from env")
}
