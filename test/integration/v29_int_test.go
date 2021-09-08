package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/stretchr/testify/suite"
)

type V29TestSuite struct {
	tagsuite.Suite
}

const rcVersion = "0.28.7-SHAf566db1"
const rcTag = "v29-update"

func (suite *V29TestSuite) installReleaseCandidate(ts *e2e.Session) {
	scriptExt := ".sh"
	shell := "bash"
	extraEnv := []string{}
	if runtime.GOOS == "windows" {
		scriptExt = ".ps1"
		shell = "powershell.exe"
		extraEnv = []string{"SHELL="}
	}

	script := filepath.Join(environment.GetRootPathUnsafe(), "installers", "install"+scriptExt)

	cp := ts.SpawnCmdWithOpts(
		shell,
		e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", "beta"),
		e2e.AppendEnv(
			fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, filepath.Join(ts.Dirs.Work, "multi-file")),
		),
		e2e.AppendEnv(extraEnv...),
	)

	expectLegacyStateToolInstallation(cp, "n")
	cp.Expect("State Tool Installed")
	cp.ExpectExitCode(0)

	suite.compareVersionedInstall(ts, filepath.Join(ts.Dirs.Work, "state"), rcVersion, suite.Equal)
}

// TestTaggedUpdateFlow is meant to test whether installing our release-candidate will update correctly
// It is supposed to not attempt an auto-update, and the tag should be set
func (suite *V29TestSuite) TestTaggedUpdateFlow() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.installReleaseCandidate(ts)

	// ensure that tagName is forwarded and stored in database
	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()
	suite.Assert().Equal(rcTag, cfg.GetString(updater.CfgUpdateTag))

	suite.Run("Tagged RC should not update", func() {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("update"),
			e2e.AppendEnv(
				fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, filepath.Join(ts.Dirs.Work, "multi-file")),
			),
		)
		cp.Expect("Updating State Tool to latest version available")
		cp.ExpectExitCode(0)
	})

	suite.Run("Update without tag", func() {
		// remote update-tag
		cfg.Set(updater.CfgUpdateTag, "")

		cp := ts.SpawnWithOpts(
			e2e.WithArgs("update"),
			e2e.AppendEnv(
				fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, filepath.Join(ts.Dirs.Work, "multi-file")),
			),
		)
		cp.Expect("Updating State Tool to latest version available")
		cp.Expect("Updating State Tool to version")
		cp.ExpectExitCode(0)

		suite.FileExists(filepath.Join(ts.Dirs.Work, "state"))
		contents, err := os.ReadFile(filepath.Join(ts.Dirs.Work, "state"))
		suite.Require().NoError(err)
		suite.Assert().Contains(contents, "#!/bin/bash")
		suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state-svc"))
		suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state-tray"))
	})

}

// TestAutoUpdateFlow meant to test whether updating from an untagged release-candidate will auto-update to a v29 update forwarding to the new State Tool.
func (suite *V29TestSuite) TestAutoUpdateFlow() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.installReleaseCandidate(ts)

	// ensure that tagName is forwarded and stored in database
	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()
	suite.Assert().Equal(rcTag, cfg.GetString(updater.CfgUpdateTag))

	// remove update tag
	cfg.Set(updater.CfgUpdateTag, "")

	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	// This should trigger the auto-update
	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"))
	cp.ExpectExitCode(0)
	suite.compareVersionedInstall(ts, filepath.Join(ts.Dirs.Work, "state"), rcVersion, suite.NotEqual)

	// after auto-update we should be still forwarded to the v29 release
	cp = ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"))
	cp.ExpectExitCode(0)
	suite.compareVersionedInstall(ts, filepath.Join(ts.Dirs.Work, "state"), rcVersion, suite.NotEqual)
}

func TestV29TestSuite(t *testing.T) {
	suite.Run(t, new(V29TestSuite))
}

func (suite *V29TestSuite) compareVersionedInstall(ts *e2e.Session, installPath, expected string, matcher matcherFunc) {
	type versionData struct {
		Version string `json:"version"`
	}

	cp := ts.SpawnCmd(installPath, "--version", "--output=json")
	cp.ExpectExitCode(0)
	actual := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &actual)

	matcher(expected, actual.Version)
}
