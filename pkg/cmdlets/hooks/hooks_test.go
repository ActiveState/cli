package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

var testhooks = []projectfile.Hook{
	projectfile.Hook{
		Name:  "firsthook",
		Value: "This is a command",
	},
	projectfile.Hook{
		Name:  "firsthook",
		Value: "This is a command also",
	},
	projectfile.Hook{
		Name:        "secondhook",
		Value:       "Believe it or not, this is a command too (not really)",
		Constraints: projectfile.Constraint{Platform: "windows", Environment: "64x"},
	},
}

func TestFilterHooks(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	// Test is limited with a filter
	filteredHooksMap, err := HashHooksFiltered(testhooks, []string{"firsthook"})
	assert.NoError(t, err, "Should not fail to filter hooks.")
	assert.Equal(t, 2, len(filteredHooksMap), "There should be two hooks in the map")

	for _, v := range filteredHooksMap {
		assert.NotEqual(t, "secondhook", v.Name, "`secondhook` should not be in the map")
	}

	// Test not limited with no filter
	filteredHooksMap, err = HashHooksFiltered(testhooks, []string{})
	assert.NoError(t, err, "Should not fail to filter hooks.")
	assert.NotNil(t, 3, len(filteredHooksMap), "There should be 2 hooks in the hooks map")

	// Test no results with non existent or set filter
	filteredHooksMap, err = HashHooksFiltered(testhooks, []string{"does_not_exist"})
	assert.NoError(t, err, "Should not fail to filter hooks.")
	assert.Zero(t, len(filteredHooksMap), "There should be zero hooks in the hook map.")
}

func TestMapHooks(t *testing.T) {
	mappedhooks, err := HashHooks(testhooks)
	assert.NoError(t, err, "Should not fail to map hooks.")
	assert.Equal(t, 3, len(mappedhooks), "There should only be 3 entries in the map")
}

func TestGetEffectiveHooks(t *testing.T) {
	project := projectfile.Project{}
	dat := strings.TrimSpace(`
name: name
owner: owner
hooks:
 - name: ACTIVATE
   value: echo Hello World!`)

	err := yaml.Unmarshal([]byte(dat), &project)
	project.Persist()
	assert.NoError(t, err, "YAML unmarshalled")

	hooks := GetEffectiveHooks("ACTIVATE")

	assert.NotZero(t, len(hooks), "Should return hooks")
}

func TestGetEffectiveHooksWithConstrained(t *testing.T) {
	project := projectfile.Project{}
	dat := strings.TrimSpace(`
name: name
owner: owner
hooks:
  - name: ACTIVATE
    value: echo Hello World
    constraints: 
        platform: foobar
        environment: foobar`)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.NoError(t, err, "YAML unmarshalled")
	project.Persist()

	hooks := GetEffectiveHooks("ACTIVATE")
	assert.Zero(t, len(hooks), "Should return no hooks")
}

func TestRunHook(t *testing.T) {
	project := projectfile.Project{}
	touch := filepath.Join(os.TempDir(), "state-test-runhook")
	os.Remove(touch)
	// Creating a file apparently leaves a hanging file handle
	// so explicitly get file and close it.  Didn't notice until
	// windows ran tests.
	cmd := "touch "
	if runtime.GOOS == "windows" {
		cmd = "cmd /c echo . > "
	}

	dat := `
name: name
owner: owner
hooks:
 - name: ACTIVATE
   value: ` + cmd + touch
	dat = strings.TrimSpace(dat)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.NoError(t, err, "YAML unmarshalled")
	project.Persist()

	err = RunHook("ACTIVATE")
	assert.NoError(t, err, "Should run hooks")
	assert.FileExists(t, touch, "Should create file as per the hook value")

	os.Remove(touch)
}

func TestRunHookFail(t *testing.T) {
	project := projectfile.Project{}
	touch := filepath.Join(os.TempDir(), "state-test-runhook")
	os.Remove(touch)

	dat := `
name: name
owner: owner
hooks:
  - name: ACTIVATE
    value: touch ` + touch + `
    constraints: 
       platform: foobar
       environment: foobar`
	dat = strings.TrimSpace(dat)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.NoError(t, err, "YAML unmarshalled")
	project.Persist()

	err = RunHook("ACTIVATE")
	assert.NoError(t, err, "Should run hooks without producing an error")

	_, err = os.Stat(touch)
	assert.Error(t, err, "Should not create file as per the constraints")

	os.Remove(touch)
}

// TestHookExists tests whether we find existing configured hooks when they are there
// and whether we don't find them if they don't exist.
func TestHookExists(t *testing.T) {
	project := projectfile.Project{}
	dat := `
name: name
owner: owner
hooks:
  - name: ACTIVATE
    value: don't touch
    constraints: 
      platform: foobar
      environment: foobar`
	dat = strings.TrimSpace(dat)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.NoError(t, err, "YAML unmarshalled")
	project.Persist()
	constraint := projectfile.Constraint{Platform: "foobar", Environment: "foobar"}
	hookExists := projectfile.Hook{Name: "ACTIVATE", Value: "don't touch", Constraints: constraint}
	hookNotExists := projectfile.Hook{Name: "ACTIVATENOT", Value: "touch", Constraints: constraint}
	exists, _ := HookExists(hookExists, &project)
	assert.True(t, exists, "Hooks should exist already.")
	Notexists, _ := HookExists(hookNotExists, &project)
	assert.False(t, Notexists, "Hooks should NOT exist already.")
}

func TestPromptOptions(t *testing.T) {
	project := projectfile.Project{}
	dat := `
name: name
owner: owner
hooks:
  - name: ACTIVATE
    value: bar
    constraints:
      platform: foobar
      environment: foobar
  - name: SOMETHING_ELSE
    value: baz`
	dat = strings.TrimSpace(dat)

	err := yaml.Unmarshal([]byte(dat), &project)
	assert.NoError(t, err, "YAML unmarshalled")
	project.Persist()
	options, optionsMap, err := PromptOptions("ACTIVATE")
	assert.NoError(t, err, "Determined options")
	assert.Equal(t, 1, len(options), "One hook returned")
	assert.Equal(t, 1, len(optionsMap), "One hook returned")
	hash, _ := project.Hooks[0].Hash()
	assert.Equal(t, hash, optionsMap[options[0]], "Hash is ACTIVATE's hash")
}
