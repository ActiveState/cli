package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"
)

type UpdateIntegrationTestSuite struct {
	tagsuite.Suite
}

type matcherFunc func(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool

// Todo https://www.pivotaltracker.com/story/show/177863116
// Update to release branch when possible
var targetBranch = "beta"
var oldUpdateVersion = "beta@0.32.2-SHA3e1d435"
var specificVersion = "0.32.2-SHA3e1d435"

func init() {
	if constants.BranchName == targetBranch {
		targetBranch = "master"
	}
}

// env prepares environment variables for the test
// disableUpdates prevents all update code from running
// testUpdate directs to the locally running update directory and requires that a test update bundles has been generated with `state run generate-test-update`
func (suite *UpdateIntegrationTestSuite) env(disableUpdates, forceUpdate bool) []string {
	env := []string{}

	if disableUpdates {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=true")
	} else {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=false")
	}

	if forceUpdate {
		env = append(env, "ACTIVESTATE_FORCE_UPDATE=true")
	}

	dir, err := ioutil.TempDir("", "system*")
	suite.NoError(err)
	env = append(env, fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir))

	return env
}

func (suite *UpdateIntegrationTestSuite) versionCompare(ts *e2e.Session, expected string, matcher matcherFunc) {
	type versionData struct {
		Version string `json:"version"`
	}

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(true, false)...))
	cp.ExpectExitCode(0)

	version := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &version)

	matcher(expected, version.Version, fmt.Sprintf("Version could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) branchCompare(ts *e2e.Session, expected string, matcher matcherFunc) {
	type branchData struct {
		Branch string `json:"branch"`
	}

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(true, false)...))
	cp.ExpectExitCode(0, 30*time.Second)

	branch := branchData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &branch)

	matcher(expected, branch.Branch, fmt.Sprintf("Branch could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) TestUpdateAvailable() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Technically state tool automatically starts the state-svc, but the update notification only happens if the svc
	// happens to already be running and fails silently if not, so in this case we want to ensure the svc is running
	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, true)...))
	cp.ExpectExitCode(0)

	// Give svc time to check for updates and cache the info
	time.Sleep(2 * time.Second)

	cp = ts.SpawnWithOpts(e2e.WithArgs("--version"))
	cp.Expect("Update Available")
	cp.ExpectExitCode(0)
}

func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.testUpdate(ts, ts.Dirs.Bin)
}

func (suite *UpdateIntegrationTestSuite) testUpdate(ts *e2e.Session, baseDir string, opts ...e2e.SpawnOptions) {
	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()

	spawnOpts := []e2e.SpawnOptions{
		e2e.WithArgs("update"),
		e2e.AppendEnv(suite.env(false, true)...),
	}
	if opts != nil {
		spawnOpts = append(spawnOpts, opts...)
	}

	cp := ts.SpawnCmdWithOpts(filepath.Join(baseDir, installation.BinDirName, constants.StateCmd+osutils.ExeExt), spawnOpts...)
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect("Installing Update")
}

func (suite *UpdateIntegrationTestSuite) TestUpdate_Repair() {
	suite.OnlyRunForTags(tagsuite.Update)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()

	subBinDir := filepath.Join(ts.Dirs.Bin, "bin")
	files, err := os.ReadDir(ts.Dirs.Bin)
	suite.NoError(err)
	for _, f := range files {
		err = fileutils.CopyFile(filepath.Join(ts.Dirs.Bin, f.Name()), filepath.Join(subBinDir, f.Name()))
		suite.NoError(err)
	}

	stateExePath := filepath.Join(ts.Dirs.Bin, filepath.Base(ts.Exe))

	spawnOpts := []e2e.SpawnOptions{
		e2e.WithArgs("update"),
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, ts.Dirs.Bin)),
		e2e.AppendEnv(suite.env(false, true)...),
	}

	cp := ts.SpawnCmdWithOpts(stateExePath, spawnOpts...)
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect("Installing Update", time.Minute)
	cp.ExpectExitCode(0)

	suite.NoFileExists(filepath.Join(ts.Dirs.Bin, constants.StateCmd+exeutils.Extension), "State Tool executable at install dir should no longer exist")
	suite.NoFileExists(filepath.Join(ts.Dirs.Bin, constants.StateSvcCmd+exeutils.Extension), "State Service executable at install dir should no longer exist")
	suite.NoFileExists(filepath.Join(ts.Dirs.Bin, constants.StateTrayCmd+exeutils.Extension), "State Tool executable at install dir should no longer exist")
}

func (suite *UpdateIntegrationTestSuite) TestUpdateChannel() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	tests := []struct {
		Name    string
		Channel string
		Version string
	}{
		{"release-channel", "release", ""},
		{"specific-update", targetBranch, specificVersion},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			// TODO: Update targetBranch and specificVersion after a v0.34.0 release
			suite.T().Skip("Skipping these tests for now as the update changes need to be available in an older version of the state tool.")
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			updateArgs := []string{"update", "--set-channel", tt.Channel}
			if tt.Version != "" {
				updateArgs = append(updateArgs, "--set-version", tt.Version)
			}
			env := []string{fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, ts.Dirs.Bin)}
			env = append(env, suite.env(false, false)...)
			cp := ts.SpawnWithOpts(
				e2e.WithArgs(updateArgs...),
				e2e.AppendEnv(env...),
			)
			cp.Expect("Updating")
			cp.ExpectExitCode(0, 1*time.Minute)

			suite.branchCompare(ts, tt.Channel, suite.Equal)

			if tt.Version != "" {
				suite.versionCompare(ts, tt.Version, suite.Equal)
			}
		})
	}
}

func (suite *UpdateIntegrationTestSuite) TestUpdateTags() {
	// Disabled, waiting for - https://www.pivotaltracker.com/story/show/179646813
	suite.T().Skip("Disabled for now")
	suite.OnlyRunForTags(tagsuite.Update)

	tests := []struct {
		name          string
		tagged        bool
		expectSuccess bool
	}{
		{"update-to-tag", false, true},
		{"update-with-tag", true, false},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()
		})
	}
}
func TestUpdateIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}
	suite.Run(t, new(UpdateIntegrationTestSuite))
}

func lockedProjectURL() string {
	return fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdate() {
	// suite.T().Skip("Test will not work until v0.34.0")
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.testAutoUpdate(ts, ts.Dirs.Bin)
}

func (suite *UpdateIntegrationTestSuite) testAutoUpdate(ts *e2e.Session, baseDir string, opts ...e2e.SpawnOptions) {
	fakeHome := filepath.Join(ts.Dirs.Work, "home")
	suite.Require().NoError(fileutils.Mkdir(fakeHome))

	spawnOpts := []e2e.SpawnOptions{
		e2e.WithArgs("--version"),
		e2e.AppendEnv(suite.env(false, true)...),
		e2e.AppendEnv(fmt.Sprintf("HOME=%s", fakeHome)),
		e2e.AppendEnv("ACTIVESTATE_TEST_AUTO_UPDATE=true"),
	}
	if opts != nil {
		spawnOpts = append(spawnOpts, opts...)
	}

	cp := ts.SpawnCmdWithOpts(filepath.Join(baseDir, "bin", constants.StateCmd+osutils.ExeExt), spawnOpts...)
	cp.Expect("Auto Update")
	cp.Expect("Updating State Tool")
	cp.Expect("Done", 5*time.Minute)
}

func (suite *UpdateIntegrationTestSuite) installLatestReleaseVersion(ts *e2e.Session, dir string) {
	var cp *termtest.ConsoleProcess
	if runtime.GOOS != "windows" {
		oneLiner := fmt.Sprintf("sh <(curl -q https://platform.activestate.com/dl/cli/pdli01/install.sh) -f -n -t %s", dir)
		cp = ts.SpawnCmdWithOpts(
			"bash", e2e.WithArgs("-c", oneLiner),
		)
	} else {
		b, err := download.GetDirect("https://platform.activestate.com/dl/cli/pdli01/install.ps1")
		suite.Require().NoError(err)

		ps1File := filepath.Join(ts.Dirs.Work, "install.ps1")
		suite.Require().NoError(fileutils.WriteFile(ps1File, b))

		cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.WithArgs(ps1File, "-f", "-n", "-t", dir),
			e2e.AppendEnv("SHELL="),
		)
	}
	cp.Expect("Installation Complete", 5*time.Minute)

	suite.FileExists(filepath.Join(dir, installation.BinDirName, constants.StateCmd+osutils.ExeExt))
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateToCurrent() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	installDir := filepath.Join(ts.Dirs.Work, "install")
	err := fileutils.MkdirUnlessExists(installDir)
	suite.NoError(err)

	suite.installLatestReleaseVersion(ts, installDir)

	suite.testAutoUpdate(ts, installDir, e2e.AppendEnv(fmt.Sprintf("ACTIVESTATE_CLI_UPDATE_BRANCH=%s", constants.BranchName)))
}

func (suite *UpdateIntegrationTestSuite) TestUpdateToCurrent() {
	if strings.HasPrefix(constants.Version, "0.30") {
		// Feel free to drop this once the release channel is no longer on 0.29
		suite.T().Skip("Updating from release 0.29 to 0.30 is not covered due to how 0.29 did updates (async)")
	}
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	installDir := filepath.Join(ts.Dirs.Work, "install")
	fileutils.MkdirUnlessExists(installDir)

	suite.installLatestReleaseVersion(ts, installDir)

	suite.testUpdate(ts, installDir, e2e.AppendEnv(fmt.Sprintf("ACTIVESTATE_CLI_UPDATE_BRANCH=%s", constants.BranchName)))
}
