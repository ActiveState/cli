package inherit

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	project := projectfile.Project{}
	project.Persist()

	Command.Execute()

	assert.True(t, true, "Execute didn't panic")
}

func TestInheritNew(t *testing.T) {
	origRecognizedVariables := recognizedVariables
	recognizedVariables = []string{"foo", "bar", "baz"}
	os.Setenv("foo", "bar")
	os.Setenv("bar", "baz")
	os.Setenv("baz", "quux")
	project := projectfile.Project{}
	project.Persist()

	Command.Execute()

	assert.Equal(t, 3, len(project.Variables), "3 env variables were inherited")
	assert.Equal(t, "foo", project.Variables[0].Name, "name is foo")
	assert.Equal(t, "bar", project.Variables[0].Value, "value is bar")
	assert.Equal(t, "bar", project.Variables[1].Name, "name is bar")
	assert.Equal(t, "baz", project.Variables[1].Value, "value is baz")
	assert.Equal(t, "baz", project.Variables[2].Name, "name is baz")
	assert.Equal(t, "quux", project.Variables[2].Value, "value is quux")

	recognizedVariables = origRecognizedVariables // restore
}

func TestOverwrite(t *testing.T) {
	origRecognizedVariables := recognizedVariables
	recognizedVariables = []string{"foo"}
	os.Setenv("foo", "baz")
	project := projectfile.Project{}
	project.Variables = append(project.Variables, projectfile.Variable{Name: "foo", Value: "bar"})
	project.Persist()

	testConfirm = true
	Command.Execute()
	assert.Equal(t, "baz", project.Variables[0].Value, "New value inherited")

	os.Setenv("foo", "bar")
	testConfirm = false
	Command.Execute()
	assert.Equal(t, "baz", project.Variables[0].Value, "New value NOT inherited")

	recognizedVariables = origRecognizedVariables // restore
}
