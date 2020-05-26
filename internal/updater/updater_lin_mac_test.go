// +build !windows

package updater

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"
)

func TestUpdaterNoError(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	updatemocks.MockUpdater(t, os.Args[0], constants.BranchName, "1.3")

	updater := createUpdater()

	err := updater.Run(testOutput(t))
	require.NoError(t, err, "Should run update")

	dir, err := ioutil.TempDir("", "state-test-updater")
	require.NoError(t, err)
	target := filepath.Join(dir, "target")
	if fileutils.FileExists(target) {
		os.Remove(target)
	}

	err = updater.Download(target)
	require.NoError(t, err)
	assert.FileExists(t, target, "Downloads to target path")

	os.Remove(target)
}
