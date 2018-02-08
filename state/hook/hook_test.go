package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/assert"
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

func TestExecute(t *testing.T) {
	// Override printOutput isn't working.  If we figure this out, use it here.
	// _ = printOutput
	// printOutput := func(hookmap map[string][]hashedhook) bool {
	// 	var expectedkeys = []string{"FIRST_INSTALL", "AFTER_UPDATE"}
	// 	var bothfound = false
	// 	for key := range hookmap {
	// 		for _, val := range expectedkeys {
	// 			if key != val {
	// 				return bothfound
	// 			}
	// 		}
	// 		bothfound = true
	// 	}
	// 	return bothfound
	// } // Error here "declared but never used"

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"hook"})

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
}

// func TestGetFilters(t *testing.T) {
// 	root, err := environment.GetRootPath()
// 	assert.NoError(t, err, "Should detect root path")
// 	os.Chdir(filepath.Join(root, "test"))
// 	var cmd = Command.GetCobraCmd()
// 	cmd.SetArgs([]string{"", "--filter", "filter1"})
// 	Flags.Filter = "filter1"
// 	assert.Equal(t, []string{"filter1"}, getFilters(cmd), "These lists of filters should be the same.")
// 	// TODO handle multiple --filters
// 	// Command.GetCobraCmd().SetArgs([]string{"--filter", "filter1", "--filter", "filter2"})
// 	cmd.SetArgs([]string{""})
// 	var emptylist []string
// 	assert.Equal(t, emptylist, getFilters(cmd), "These lists of filters should be the same.")

// }

// func TestFilterHooks(t *testing.T) {
// 	root, err := environment.GetRootPath()
// 	assert.NoError(t, err, "Should detect root path")
// 	os.Chdir(filepath.Join(root, "test"))
// 	var cmd = Command.GetCobraCmd()
// 	// Test is limited with a filter
// 	cmd.SetArgs([]string{"", "--filter", "FIRST_INSTALL"})
// 	filteredHooksMap := filterHooks(getFilters(cmd))
// 	assert.Equal(t, 1, len(filteredHooksMap), "There should be only one hook in the map")
// 	assert.Equal(t, []string{}, filteredHooksMap["AFTER_UPDATE"], "`AFTER_UPDATE` should not be in the map so this should be an empty list")

// 	// Test not limited with no filter
// 	cmd.SetArgs([]string{""})
// 	filteredHooksMap = filterHooks(getFilters(cmd))
// 	assert.NotNil(t, 2, len(filteredHooksMap), "There should be 2 hooks in the hooks map")

// 	// Test no results with non existent or set filter
// 	cmd.SetArgs([]string{"", "--filter", "does_not_exist"})
// 	filteredHooksMap = filterHooks(getFilters(cmd))
// 	assert.Equal(t, 0, len(filteredHooksMap), "There should be zero hooks in the hook map.  None found by filter name.")
// }

func TestHashHookStruct(t *testing.T) {
	binHash, _ := hashstructure.Hash(testhooks[0], nil)
	expected := fmt.Sprintf("%X", binHash)
	actual := HashHookStruct(testhooks[0])
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
	mappedhooks := MapHooks(testhooks)
	assert.True(t, checkMapKeys(mappedhooks, keys), fmt.Sprintf("Map should have keys '%v' and '%v' but does not: %v", keys[0], keys[1], mappedhooks))
	assert.Equal(t, 2, len(mappedhooks), "There should only be 2 triggers/keys in the map")
	assert.Equal(t, 2, len(mappedhooks["firsthook"]), "There should be 2 commands for the `firsthook` hook")
}
