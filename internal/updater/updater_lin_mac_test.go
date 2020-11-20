// +build !windows

package updater

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"
)

func TestUpdaterNoError(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	updatemocks.MockUpdater(t, os.Args[0], constants.BranchName, "1.3")

	updater := createUpdater()

	out := outputhelper.NewCatcher()
	err := updater.Run(out.Outputer, false)
	require.NoError(t, err, "Should run update")
	// should notify about update_attempt
	assert.NotEqual(t, "", strings.TrimSpace(out.CombinedOutput()))

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
