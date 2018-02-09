package hook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/stretchr/testify/assert"
)

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
