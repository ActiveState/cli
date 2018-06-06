package variables

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

// This is mostly a clone of the hook cmdlet's tests. Any tests added, modified,
// or removed in that file should be applied here and vice-versa.

var testvariables = []projectfile.Variable{
	projectfile.Variable{
		Name:  "foo",
		Value: "bar",
	},
	projectfile.Variable{
		Name:  "bar",
		Value: "baz",
	},
	projectfile.Variable{
		Name:        "bar",
		Value:       "quux",
		Constraints: projectfile.Constraint{Platform: "windows"},
	},
}

func TestFilterVariables(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	// Test is limited with a filter
	filteredVariablesMap, err := HashVariablesFiltered(testvariables, []string{"foo"})
	assert.NoError(t, err, "Should not fail to filter variables.")
	assert.Equal(t, 1, len(filteredVariablesMap), "There should be two variables in the map")

	for _, v := range filteredVariablesMap {
		assert.NotEqual(t, "bar", v.Name, "`bar` should not be in the map")
	}

	// Test not limited with no filter
	filteredVariablesMap, err = HashVariablesFiltered(testvariables, []string{})
	assert.NoError(t, err, "Should not fail to filter variables.")
	assert.NotNil(t, 3, len(filteredVariablesMap), "There should be 2 variables in the variables map")

	// Test no results with non existent or set filter
	filteredVariablesMap, err = HashVariablesFiltered(testvariables, []string{"does_not_exist"})
	assert.NoError(t, err, "Should not fail to filter variables.")
	assert.Zero(t, len(filteredVariablesMap), "There should be zero variables in the variable map.")
}

func TestMapVariables(t *testing.T) {
	mappedvariables, err := HashVariables(testvariables)
	assert.NoError(t, err, "Should not fail to map variables.")
	assert.Equal(t, 3, len(mappedvariables), "There should only be 3 entries in the map")
}

func TestGetEffectiveVariables(t *testing.T) {
	project := projectfile.Project{}
	dat := strings.TrimSpace(`
name: name
owner: owner
variables:
 - name: foo
   value: bar`)

	err := yaml.Unmarshal([]byte(dat), &project)
	project.Persist()
	assert.NoError(t, err, "YAML unmarshalled")

	variables := GetEffectiveVariables()

	assert.NotZero(t, len(variables), "Should return variables")
}

func TestGetEffectiveVariablesWithConstrained(t *testing.T) {
	project := projectfile.Project{}
	dat := strings.TrimSpace(`
name: name
owner: owner
variables:
  - name: foo
    value: bar
    constraints:
        platform: foobar
        environment: foobar`)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.NoError(t, err, "YAML unmarshalled")
	project.Persist()

	variables := GetEffectiveVariables()
	assert.Zero(t, len(variables), "Should return no variables")
}

// TestVariableExists tests whether we find existing defined variables when they are there
// and whether we don't find them if they don't exist.
func TestVariableExists(t *testing.T) {
	project := projectfile.Project{}
	dat := `
name: name
owner: owner
variables:
  - name: foo
    value: bar
    constraints:
      platform: foobar
      environment: foobar`
	dat = strings.TrimSpace(dat)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.NoError(t, err, "YAML unmarshalled")
	project.Persist()
	constraint := projectfile.Constraint{Platform: "foobar", Environment: "foobar"}
	variableExists := projectfile.Variable{Name: "foo", Value: "bar", Constraints: constraint}
	variableNotExists := projectfile.Variable{Name: "bar", Value: "baz", Constraints: constraint}
	exists, _ := VariableExists(variableExists, &project)
	assert.True(t, exists, "Variables should exist already.")
	Notexists, _ := VariableExists(variableNotExists, &project)
	assert.False(t, Notexists, "Variables should NOT exist already.")
}
