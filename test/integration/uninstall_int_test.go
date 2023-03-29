package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type UninstallIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *UninstallIntegrationTestSuite) TestUninstall() {
	suite.OnlyRunForTags(tagsuite.Uninstall, tagsuite.Critical)
	suite.T().Run("Partial uninstall", func(t *testing.T) { suite.testUninstall(false) })
	suite.T().Run("Full uninstall", func(t *testing.T) { suite.testUninstall(true) })
}

func (suite *UninstallIntegrationTestSuite) testUninstall(all bool) {
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	mockBranchDir := filepath.Join(ts.Dirs.Work, "StateTool", constants.BranchName)
	mockBinDir := filepath.Join(mockBranchDir, "bin")
	err := fileutils.Mkdir(mockBinDir)
	suite.NoError(err)

	ts.Exe = ts.CopyExeToDir(ts.Exe, mockBinDir)
	ts.SvcExe = ts.CopyExeToDir(ts.SvcExe, mockBinDir)
	ts.Dirs.Bin = mockBinDir

	defaultMarker := filepath.Join(filepath.Dir(ts.Dirs.Work), installation.InstallDirMarker)
	err = fileutils.CopyFile(defaultMarker, filepath.Join(mockBranchDir, installation.InstallDirMarker))
	suite.NoError(err)

	err = os.Remove(filepath.Join(defaultMarker))
	suite.NoError(err)

	isAdmin, err := osutils.IsAdmin()
	suite.NoError(err)

	err = installation.SaveContext(&installation.Context{InstalledAsAdmin: isAdmin})
	suite.NoError(err)

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"))
	cp.ExpectExitCode(0)

	if all {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("clean", "uninstall", "--all"),
			e2e.WithWorkDirectory(mockBinDir),
		)
	} else {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("clean", "uninstall"),
			e2e.WithWorkDirectory(mockBinDir),
		)
	}
	cp.Expect("You are about to remove")
	if !all {
		cp.Expect("--all") // verify mention of "--all" to remove everything
	}
	cp.SendLine("y")
	if runtime.GOOS == "windows" {
		cp.ExpectLongString("Deletion of State Tool has been scheduled.")
	} else {
		cp.ExpectLongString("Successfully removed State Tool and related files")
	}
	cp.ExpectExitCode(0)

	if runtime.GOOS == "windows" {
		// Allow time for spawned script to remove directories
		time.Sleep(500 * time.Millisecond)
	}

	if all {
		suite.NoDirExists(ts.Dirs.Cache, "Cache dir should not exist after full uninstall")
		suite.NoDirExists(ts.Dirs.Config, "Config dir should not exist after full uninstall")
	} else {
		suite.DirExists(ts.Dirs.Cache, "Cache dir should still exist after partial uninstall")
		suite.DirExists(ts.Dirs.Config, "Config dir should still exist after partial uninstall")
	}

	if fileutils.FileExists(ts.Exe) {
		suite.Fail("State tool executable should not exist after uninstall")
	}

	if fileutils.FileExists(ts.SvcExe) {
		suite.Fail("State service executable should not exist after uninstall")
	}

	if runtime.GOOS == "linux" {
		// When installed in a non-desktop environment (i.e. on a server), verify the user's ~/.profile was reverted.
		homeDir, err := user.HomeDir()
		suite.Require().NoError(err)
		profile := filepath.Join(homeDir, ".profile")
		suite.NotContains(string(fileutils.ReadFileUnsafe(profile)), ts.SvcExe, "autostart should not be configured for Linux server environment anymore")
	}

	if fileutils.DirExists(ts.Dirs.Bin) {
		suite.Fail("bin directory should not exist after uninstall")
	}
}

func TestUninstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UninstallIntegrationTestSuite))
}
