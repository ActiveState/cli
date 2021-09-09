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

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type V29TestSuite struct {
	tagsuite.Suite
}

type versionData struct {
	Version string `json:"version"`
}

const rcVersion = "0.28.7-SHAf566db1"
const rcTag = "v29-update"

func (suite *V29TestSuite) installReleaseCandidate(ts *e2e.Session) string {
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

	cp.Expect("proceed with install?")
	cp.SendLine("Y")
	cp.Expect(fmt.Sprintf("Fetching the latest version: %s", rcVersion))
	if runtime.GOOS == "windows" {
		cp.Expect("successfully installed to", 30*time.Second)
	} else {
		cp.Expect("Installing to", 30*time.Second)
		cp.Expect("Allow $PATH to be appended in your")
		cp.SendLine("n")
		cp.Expect("State Tool installation complete")
	}
	cp.ExpectExitCode(0)

	stateExe := filepath.Join(ts.Dirs.Work, "state")
	suite.compareVersionedInstall(ts, stateExe, rcVersion, suite.Equal)

	return stateExe
}

// TestTaggedUpdateFlow is meant to test whether installing our release-candidate will update correctly
// It is supposed to not attempt an auto-update, and the tag should be set
func (suite *V29TestSuite) TestTaggedUpdateFlow() {
	suite.OnlyRunForTags(tagsuite.V29Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	stateExe := suite.installReleaseCandidate(ts)

	// ensure tag is in config yaml file
	f, err := os.ReadFile(filepath.Join(ts.Dirs.Config, "config.yaml"))
	suite.Require().NoError(err)
	suite.Require().Contains(string(f), fmt.Sprintf(`tag: %s`, rcTag))

	suite.Run("Tagged RC should not update", func() {
		cp := ts.SpawnCmdWithOpts(
			stateExe,
			e2e.WithArgs("update"),
			e2e.AppendEnv(
				fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, filepath.Join(ts.Dirs.Work, "multi-file")),
			),
		)
		cp.Expect("You are already using the latest State Tool version")
		cp.ExpectExitCode(0)
	})

	suite.Run("Update without tag", func() {
		// remove update-tag
		err := os.Remove(filepath.Join(ts.Dirs.Config, "config.yaml"))
		suite.Require().NoError(err)

		cp := ts.SpawnCmdWithOpts(
			stateExe,
			e2e.WithArgs("update"),
			e2e.AppendEnv(
				fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, filepath.Join(ts.Dirs.Work, "multi-file")),
			),
		)
		cp.Expect("Updating State Tool to latest version available")
		cp.Expect("Version updated to", 60*time.Second)
		cp.ExpectExitCode(0)

		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".bat"
		}
		suite.Assert().FileExists(filepath.Join(ts.Dirs.Work, "state"+ext), "Transitional state tool script does not exist.")
		suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state-svc"))
		suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state-tray"))
	})

}

// TestAutoUpdateFlow meant to test whether updating from an untagged release-candidate will auto-update to a v29 update forwarding to the new State Tool.
func (suite *V29TestSuite) TestAutoUpdateFlow() {
	suite.OnlyRunForTags(tagsuite.V29Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	stateExe := suite.installReleaseCandidate(ts)

	// ensure that tagName is forwarded and stored in database
	// ensure tag is in config yaml file
	f, err := os.ReadFile(filepath.Join(ts.Dirs.Config, "config.yaml"))
	suite.Require().NoError(err)
	suite.Require().Contains(string(f), fmt.Sprintf(`tag: %s`, rcTag))

	// remove update tag
	err = os.Remove(filepath.Join(ts.Dirs.Config, "config.yaml"))
	suite.Require().NoError(err)

	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(stateExe, t, t)

	// This should trigger the auto-update
	cp := ts.SpawnCmdWithOpts(
		stateExe,
		e2e.WithArgs("--version", "--output=json"),
		e2e.AppendEnv(fmt.Sprintf("%s=false", constants.DisableUpdates)))
	cp.ExpectExitCode(0, 60*time.Second)
	actual := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &actual)
	suite.NotEqual(rcVersion, actual.Version, "Version should have changed due to auto-update")

	// after auto-update we should be still forwarded to the v29 release
	suite.compareVersionedInstall(ts, stateExe, rcVersion, suite.NotEqual)
}

func TestV29TestSuite(t *testing.T) {
	suite.Run(t, new(V29TestSuite))
}

func (suite *V29TestSuite) compareVersionedInstall(ts *e2e.Session, installPath, expected string, matcher matcherFunc) {
	cp := ts.SpawnCmd(installPath, "--version", "--output=json")
	cp.ExpectExitCode(0)
	actual := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &actual)

	matcher(expected, actual.Version)
}
