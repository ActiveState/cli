package remove

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

// This is mostly a clone of the state/hooks/remove/remove_test.go file. Any
// tests added, modified, or removed in that file should be applied here and
// vice-versa.

// For moving the CWD when needed during a test.
var startingDir string
var tempDir string

func setup(t *testing.T) {
	err := moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

	Args.Identifier = ""
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	projectfile.Reset()
}

func teardown() {
	removeTmpDir()
}

// Moves process into a tmp dir and brings a copy of project file with it
func moveToTmpDir() error {
	var err error
	startingDir, _ = environment.GetRootPath()
	startingDir = filepath.Join(startingDir, "test")
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

func TestExecute(t *testing.T) {
	setup(t)
	defer teardown()

	assert := assert.New(t)
	Command.Execute()
	assert.Equal(true, true, "Execute didn't panic")
}

func TestRemoveByHashCmd(t *testing.T) {
	setup(t)
	defer teardown()

	project := projectfile.Get()
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
	defer teardown()

	project := projectfile.Get()
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
	defer teardown()

	options, optionsMap, err := variables.PromptOptions("")
	print.Formatted("\nmap1: %v\n", optionsMap)
	assert.NoError(t, err, "Should be able to get prompt options")

	testPromptResultOverride = options[0]

	removed := removeByPrompt("")
	assert.NotNil(t, removed, "Received a removed variable")

	hash, _ := removed.Hash()
	assert.Equal(t, optionsMap[testPromptResultOverride], hash, "Should have removed one variable")
}

func TestRemoveByHash(t *testing.T) {
	setup(t)
	defer teardown()

	project := projectfile.Get()
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
	defer teardown()

	project := projectfile.Get()
	variableLen := len(project.Variables)

	removed := removeByName(project.Variables[0].Name)
	assert.NotNil(t, removed, "Received a removed variable")

	assert.Equal(t, variableLen-1, len(project.Variables), "One variable should have been removed")
}

// This test shoudln't remove anything as there are multiple variables defined for the same variable name
func TestRemoveByNameFailCmd(t *testing.T) {
	setup(t)
	defer teardown()
	testPromptResultOverride = "" // reset

	varName := "foo"
	project := projectfile.Get()

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
