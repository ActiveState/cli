package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

var testhooks = []projectfile.Hook{
	projectfile.Hook{
		"firsthook",
		"This is a command",
		projectfile.Constraint{"windows", "64x"},
	},
	projectfile.Hook{
		"firsthook",
		"This is a command also",
		projectfile.Constraint{"windows", "64x"},
	},
	projectfile.Hook{
		"secondhook",
		"Believe it or not, this is a command too (not really)",
		projectfile.Constraint{"windows", "64x"},
	},
}

func TestFilterHooks(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	// Test is limited with a filter
	filteredHooksMap, err := FilterHooks([]string{"FIRST_INSTALL"})
	assert.NoError(t, err, "Should not fail to filter hooks.")
	assert.Equal(t, 1, len(filteredHooksMap), "There should be only one hook in the map")
	assert.Equal(t, 0, len(filteredHooksMap["AFTER_UPDATE"]), "`AFTER_UPDATE` should not be in the map so this should be an empty list")

	// Test not limited with no filter
	filteredHooksMap, err = FilterHooks([]string{})
	assert.NoError(t, err, "Should not fail to filter hooks.")
	assert.NotNil(t, 2, len(filteredHooksMap), "There should be 2 hooks in the hooks map")

	// Test no results with non existent or set filter
	filteredHooksMap, err = FilterHooks([]string{"does_not_exist"})
	assert.NoError(t, err, "Should not fail to filter hooks.")
	assert.Nil(t, filteredHooksMap, "There should be zero hooks in the hook map.")
}

func TestHashHookStruct(t *testing.T) {
	binHash, _ := hashstructure.Hash(testhooks[0], nil)
	expected := fmt.Sprintf("%X", binHash)
	actual, err := HashHookStruct(testhooks[0])
	assert.NoError(t, err, "Should not fail to hash hook struct.")
	assert.Equal(t, expected, actual, "The hash of the same struct should be the same")
}

func checkMapKeys(mappedhooks map[string][]Hashedhook, keys []string) bool {
	numFound := 0
	for key := range mappedhooks {
		for _, expectedKey := range keys {
			if key == expectedKey {
				numFound++
			}
		}
	}
	if numFound != len(keys) {
		return false
	}
	return true
}
func TestMapHooks(t *testing.T) {
	keys := []string{"firsthook", "secondhook"}
	mappedhooks, err := MapHooks(testhooks)
	assert.NoError(t, err, "Should not fail to map hooks.")
	assert.True(t, checkMapKeys(mappedhooks, keys), fmt.Sprintf("Map should have keys '%v' and '%v' but does not: %v", keys[0], keys[1], mappedhooks))
	assert.Equal(t, 2, len(mappedhooks), "There should only be 2 triggers/keys in the map")
	assert.Equal(t, 2, len(mappedhooks["firsthook"]), "There should be 2 commands for the `firsthook` hook")
}

func TestGetEffectiveHooks(t *testing.T) {
	project := projectfile.Project{}
	dat := strings.TrimSpace(`
name: name
owner: owner
hooks:
 - name: ACTIVATE
   value: echo Hello World!`)

	yaml.Unmarshal([]byte(dat), &project)

	hooks := GetEffectiveHooks("ACTIVATE", &project)

	assert.NotZero(t, len(hooks), "Should return hooks")
}

func TestGetEffectiveHooksWithConstrained(t *testing.T) {
	project := projectfile.Project{}
	dat := strings.TrimSpace(`
name: name
owner: owner
hooks:
 - name: ACTIVATE
   value: echo Hello World!
   constraints: 
	- platform: foobar
	  environment: foobar`)

	yaml.Unmarshal([]byte(dat), &project)

	hooks := GetEffectiveHooks("ACTIVATE", &project)
	assert.Zero(t, len(hooks), "Should return no hooks")
}

func TestRunHook(t *testing.T) {
	project := projectfile.Project{}
	touch := filepath.Join(os.TempDir(), "state-test-runhook")
	os.Remove(touch)

	dat := `
name: name
owner: owner
hooks:
 - name: ACTIVATE
   value: touch ` + touch
	dat = strings.TrimSpace(dat)

	yaml.Unmarshal([]byte(dat), &project)

	err := RunHook("ACTIVATE", &project)
	assert.NoError(t, err, "Should run hooks")
	assert.FileExists(t, touch, "Should create file as per the hook value")
}
