package add

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/pkg/cmdlets/variables"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

// This is mostly a clone of the state/hooks/add/add_test.go file. Any tests
// added, modified, or removed in that file should be applied here and
// vice-versa.

// For moving the CWD when needed during a test.
var startingDir string
var tempDir string

// Moves process into a tmp dir and brings a copy of project file with it
func moveToTmpDir() error {
	var err error
	root, err := environment.GetRootPath()
	testDir := filepath.Join(root, "test")
	os.Chdir(testDir)
	if err != nil {
		return err
	}
	startingDir, _ = os.Getwd()
	tempDir, err = ioutil.TempDir("", "CLI-")
	if err != nil {
		return err
	}
	err = os.Chdir(tempDir)
	if err != nil {
		return err
	}

	copy(filepath.Join(startingDir, "activestate.yaml"),
		filepath.Join(tempDir, "activestate.yaml"))
	return nil
}

// Moves process to original dir and deletes temp
func removeTmpDir() error {
	err := os.Chdir(startingDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(tempDir)
	if err != nil {
		return err
	}
	return nil
}

func copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	in.Close()
	return out.Close()
}

func TestAddVariablePass(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	err := moveToTmpDir()

	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

	newVariableName := "foo"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{newVariableName, "bar"})
	Cc.Execute()

	project := projectfile.Get()
	var found = false
	for _, variable := range project.Variables {
		if variable.Name == newVariableName {
			found = true
		}
	}
	assert.True(t, found, fmt.Sprintf("Should find a variable named %v", newVariableName))

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}

func TestAddVariableFail(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	err := moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

	Cc := Command.GetCobraCmd()
	newVariableName := "foo?!"
	Cc.SetArgs([]string{newVariableName})
	Cc.Execute()
	project := projectfile.Get()

	var found = false
	for _, variable := range project.Variables {
		if variable.Name == newVariableName {
			found = true
		}
	}
	assert.False(t, found, fmt.Sprintf("Should NOT find a variable named %v", newVariableName))
	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}

// Test it doesn't explode when run with no args
func TestExecute(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Command.Execute()

	assert.Equal(t, true, true, "Execute didn't panic")
}

//
func TestAddVariableFailIdentical(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	project := projectfile.Get()
	err := moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

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

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}

func TestAddVariableInheritValue(t *testing.T) {
	Args.Name, Args.Value = "", "" // reset
	err := moveToTmpDir()

	assert.Nil(t, err, "A temporary directory was created and entered as CWD")
	os.Setenv("foo", "baz")

	newVariableName := "foo"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{newVariableName})
	Cc.Execute()

	project := projectfile.Get()
	var found *projectfile.Variable
	for _, variable := range project.Variables {
		if variable.Name == newVariableName {
			found = &variable
		}
	}
	assert.NotNil(t, found, fmt.Sprintf("Should find a variable named %v", newVariableName))
	assert.Equal(t, os.Getenv("foo"), found.Value, "Variable value should be inherited from env")

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}
