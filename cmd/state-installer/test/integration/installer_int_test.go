package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type InstallerIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InstallerIntegrationTestSuite) TestInstallFromInstallationError() {
	suite.OnlyRunForTags(tagsuite.Installer)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnCmdWithOpts(
		ts.InstallerExe,
		e2e.WithArgs(filepath.Join(ts.Dirs.Work, "installation"), "--source-path", ts.Dirs.Base),
		e2e.AppendEnv(constants.DisableUpdates+"=false"))

	cp.Expect("Cannot run state-installer from an installation directory.")
	cp.ExpectExitCode(1)
}

func (suite *InstallerIntegrationTestSuite) TestInstallFromLocalSource() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Determine URL of installer archive.
	baseUrl := "https://state-tool.s3.amazonaws.com/update/state"
	archiveExt := "tar.gz"
	if runtime.GOOS == "windows" {
		archiveExt = "zip"
	}
	archiveUrl := fmt.Sprintf("%s/%s/%s/%s-amd64/state-%s-amd64-%s.%s",
		baseUrl,
		constants.BranchName,
		constants.Version,
		runtime.GOOS,
		runtime.GOOS,
		constants.Version,
		archiveExt)

	// Fetch it.
	b, err := download.GetDirect(archiveUrl)
	suite.Require().NoError(err)

	// Extract it.
	installerDir := filepath.Join(ts.Dirs.Work, "installer")
	if runtime.GOOS != "windows" {
		suite.Require().NoError(unarchiver.NewTarGzBlob(b).Unarchive(installerDir))
	} else {
		suite.Require().NoError(unarchiver.NewZipBlob(b).Unarchive(installerDir))
	}

	target := filepath.Join(ts.Dirs.Work, "installation")

	// Run installer with source-path flag (ie. install from this local path)
	cp := ts.SpawnCmdWithOpts(
		filepath.Join(installerDir, constants.ToplevelInstallArchiveDir, constants.StateInstallerCmd+osutils.ExeExt),
		e2e.WithArgs(target, "--source-path", installerDir),
		e2e.AppendEnv(constants.DisableUpdates+"=false"))

	// Assert output
	cp.Expect("Installing State Tool")
	cp.Expect("Done")
	cp.Expect("successfully installed")

	// Assert expected files were installed (note this didn't use an update payload, so there's no bin directory)
	suite.FileExists(appinfo.StateApp(target).Exec())
	suite.FileExists(appinfo.SvcApp(target).Exec())

	// Assert that the config was written (ie. RC files or windows registry)
	suite.AssertConfig(ts)
}

func (suite *InstallerIntegrationTestSuite) AssertConfig(ts *e2e.Session) {
	if runtime.GOOS != "windows" {
		// Test bashrc
		homeDir, err := os.UserHomeDir()
		suite.Require().NoError(err)

		fname := ".bashrc"
		if strings.Contains(os.Getenv("SHELL"), "zsh") {
			fname = ".zshrc"
		}

		bashContents := fileutils.ReadFileUnsafe(filepath.Join(homeDir, fname))
		suite.Contains(string(bashContents), constants.RCAppendInstallStartLine, "rc file should contain our RC Append Start line")
		suite.Contains(string(bashContents), constants.RCAppendInstallStopLine, "rc file should contain our RC Append Stop line")
		suite.Contains(string(bashContents), filepath.Join(ts.Dirs.Work), "rc file should contain our target dir")
	} else {
		// Test registry
		out, err := exec.Command("reg", "query", `HKLM\SYSTEM\ControlSet001\Control\Session Manager\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)

		// we need to look for  the short and the long version of the target PATH, because Windows translates between them arbitrarily
		shortPath, err := fileutils.GetShortPathName(ts.Dirs.Work)
		suite.Require().NoError(err)
		longPath, err := fileutils.GetLongPathName(ts.Dirs.Work)
		suite.Require().NoError(err)
		if !strings.Contains(string(out), shortPath) && !strings.Contains(string(out), longPath) && !strings.Contains(string(out), ts.Dirs.Work) {
			suite.T().Errorf("registry PATH \"%s\" does not contain \"%s\", \"%s\" or \"%s\"", out, ts.Dirs.Work, shortPath, longPath)
		}
	}
}

func TestInstallerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerIntegrationTestSuite))
}
