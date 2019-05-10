package update

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func testdir(t *testing.T) string {
	cwd, err := environment.GetRootPath()
	require.NoError(t, err, "Should fetch cwd")
	testdatadir := filepath.Join(cwd, "state", "update", "testdata")
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	fileutils.CopyFile(filepath.Join(testdatadir, constants.ConfigFileName), filepath.Join(tempDir, constants.ConfigFileName))
	return tempDir
}

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	updatemocks.MockUpdater(t, os.Args[0], constants.BranchName, "1.2.3-123")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"update"})

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
	assert.NoError(failures.Handled(), "No failure occurred")
}

func TestLock(t *testing.T) {
	assert := assert.New(t)

	testdir := testdir(t)
	os.Chdir(testdir)

	projectfile.Reset()
	pjfile := projectfile.Get()
	assert.Empty(pjfile.Version, "Version is not set")
	assert.Empty(pjfile.Branch, "Branch is not set")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"update", "--lock"})

	out := capturer.CaptureStdout(func() {
		Command.Execute()
	})

	assert.Contains(out, "Version locked")

	projectfile.Reset()
	pjfile = projectfile.Get()
	assert.NotEmpty(pjfile.Version, "Version is set")
	assert.NotEmpty(pjfile.Branch, "Branch is set")
}
