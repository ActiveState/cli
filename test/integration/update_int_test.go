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

	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type UpdateIntegrationTestSuite struct {
	tagsuite.Suite
}

type matcherFunc func(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool

// Todo https://www.pivotaltracker.com/story/show/177863116
// Update to release channel when possible
var (
	targetChannel    = "beta"
	oldUpdateVersion = "beta@0.32.2-SHA3e1d435"
)

func init() {
	if constants.ChannelName == targetChannel {
		targetChannel = "master"
	}
}

// env prepares environment variables for the test
// disableUpdates prevents all update code from running
// testUpdate directs to the locally running update directory and requires that a test update bundles has been generated with `state run generate-test-update`
func (suite *UpdateIntegrationTestSuite) env(disableUpdates, forceUpdate bool) []string {
	env := []string{}

	if disableUpdates {
		env = append(env, constants.DisableUpdates+"=true")
	} else {
		env = append(env, constants.DisableUpdates+"=false")
	}

	if forceUpdate {
		env = append(env, constants.TestAutoUpdateEnvVarName+"=true")
		env = append(env, constants.ForceUpdateEnvVarName+"=true")
	}

	dir, err := os.MkdirTemp("", "system*")
	suite.NoError(err)
	env = append(env, fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir))

	return env
}

func (suite *UpdateIntegrationTestSuite) versionCompare(ts *e2e.Session, expected string, matcher matcherFunc) {
	type versionData struct {
		Version string `json:"version"`
	}

	cp := ts.SpawnWithOpts(e2e.OptArgs("--version", "--output=json"), e2e.OptAppendEnv(suite.env(true, false)...))
	cp.ExpectExitCode(0)

	version := versionData{}
	out := cp.StrippedSnapshot()
	json.Unmarshal([]byte(out), &version)

	matcher(expected, version.Version, fmt.Sprintf("Version could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) channelCompare(ts *e2e.Session, expected string, matcher matcherFunc) {
	type channelData struct {
		Channel string `json:"channel"`
	}

	cp := ts.SpawnWithOpts(e2e.OptArgs("--version", "--output=json"), e2e.OptAppendEnv(suite.env(true, false)...))
	cp.ExpectExitCode(0, termtest.OptExpectTimeout(30*time.Second))

	channel := channelData{}
	out := cp.StrippedSnapshot()
	json.Unmarshal([]byte(out), &channel)

	matcher(expected, channel.Channel, fmt.Sprintf("Channel could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) TestUpdateAvailable() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()
	cfg.Set(constants.AutoUpdateConfigKey, "false")

	search, found := "Update Available", false
	for i := 0; i < 4; i++ {
		if i > 0 {
			time.Sleep(time.Second * 3)
		}

		cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(suite.env(false, true)...))
		cp.ExpectExitCode(0)

		if strings.Contains(cp.Snapshot(), search) {
			found = true
			break
		}
	}

	suite.Require().True(found, "Expecting to find %q", search)
}

func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.testUpdate(ts, filepath.Dir(ts.Dirs.Bin))
}

func (suite *UpdateIntegrationTestSuite) testUpdate(ts *e2e.Session, baseDir string, opts ...e2e.SpawnOptSetter) {
	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()

	spawnOpts := []e2e.SpawnOptSetter{
		e2e.OptArgs("update"),
		e2e.OptAppendEnv(suite.env(false, true)...),
	}
	if opts != nil {
		spawnOpts = append(spawnOpts, opts...)
	}

	stateExec, err := installation.StateExecFromDir(baseDir)
	suite.NoError(err)

	searchA, searchB, found := "Updating State Tool to", "Installing Update", false
	for i := 0; i < 4; i++ {
		if i > 0 {
			time.Sleep(time.Second * 3)
		}

		cp := ts.SpawnCmdWithOpts(stateExec, spawnOpts...)
		cp.ExpectExitCode(0)

		snap := cp.Snapshot()
		if strings.Contains(snap, searchA) && strings.Contains(snap, searchB) {
			found = true
			break
		}
	}

	suite.Require().True(found, "Expecting to find %q and %q", searchA, searchB)
}

func (suite *UpdateIntegrationTestSuite) TestUpdate_Repair() {
	suite.OnlyRunForTags(tagsuite.Update)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()

	subBinDir := filepath.Join(ts.Dirs.Bin, "bin")
	files, err := ioutil.ReadDir(ts.Dirs.Bin)
	suite.NoError(err)
	for _, f := range files {
		err = fileutils.CopyFile(filepath.Join(ts.Dirs.Bin, f.Name()), filepath.Join(subBinDir, f.Name()))
		suite.NoError(err)
	}

	stateExePath := filepath.Join(ts.Dirs.Bin, filepath.Base(ts.Exe))

	spawnOpts := []e2e.SpawnOptSetter{
		e2e.OptArgs("update"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, ts.Dirs.Bin)),
		e2e.OptAppendEnv(suite.env(false, true)...),
	}

	searchA, searchB, found := "Updating State Tool to version", "Installing Update", false
	for i := 0; i < 4; i++ {
		if i > 0 {
			time.Sleep(time.Second * 3)
		}

		cp := ts.SpawnCmdWithOpts(stateExePath, spawnOpts...)
		cp.ExpectExitCode(0, termtest.OptExpectTimeout(time.Minute))

		snap := cp.Snapshot()
		if strings.Contains(snap, searchA) && strings.Contains(snap, searchB) {
			found = true
			break
		}
	}

	suite.Require().True(found, "Expecting to find %q and %q", searchA, searchB)

	suite.NoFileExists(filepath.Join(ts.Dirs.Bin, constants.StateCmd+osutils.ExeExtension), "State Tool executable at install dir should no longer exist")
	suite.NoFileExists(filepath.Join(ts.Dirs.Bin, constants.StateSvcCmd+osutils.ExeExtension), "State Service executable at install dir should no longer exist")
}

func (suite *UpdateIntegrationTestSuite) TestUpdateChannel() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	tests := []struct {
		Name    string
		Channel string
	}{
		{"release-channel", "release"},
		{"specific-update", targetChannel},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			// TODO: Update targetChannel and specificVersion after a v0.34.0 release
			suite.T().Skip("Skipping these tests for now as the update changes need to be available in an older version of the state tool.")
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			updateArgs := []string{"update", "--set-channel", tt.Channel}
			env := []string{fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, ts.Dirs.Bin)}
			env = append(env, suite.env(false, false)...)
			cp := ts.SpawnWithOpts(
				e2e.OptArgs(updateArgs...),
				e2e.OptAppendEnv(env...),
			)
			cp.Expect("Updating")
			cp.ExpectExitCode(0, termtest.OptExpectTimeout(1*time.Minute))

			suite.channelCompare(ts, tt.Channel, suite.Equal)
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
	return fmt.Sprintf("https://%s/string/string", constants.PlatformURL)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdate() {
	// suite.T().Skip("Test will not work until v0.34.0")
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.testAutoUpdate(ts, filepath.Dir(ts.Dirs.Bin))
}

func (suite *UpdateIntegrationTestSuite) testAutoUpdate(ts *e2e.Session, baseDir string, opts ...e2e.SpawnOptSetter) {
	fakeHome := filepath.Join(ts.Dirs.Work, "home")
	suite.Require().NoError(fileutils.Mkdir(fakeHome))

	spawnOpts := []e2e.SpawnOptSetter{
		e2e.OptArgs("--version"),
		e2e.OptAppendEnv(suite.env(false, true)...),
		e2e.OptAppendEnv(fmt.Sprintf("HOME=%s", fakeHome)),
	}
	if opts != nil {
		spawnOpts = append(spawnOpts, opts...)
	}

	stateExec, err := installation.StateExecFromDir(baseDir)
	suite.NoError(err)

	search, found := "Updating State Tool", false
	for i := 0; i < 4; i++ {
		if i > 0 {
			time.Sleep(time.Second * 4)
		}

		cp := ts.SpawnCmdWithOpts(stateExec, spawnOpts...)
		cp.ExpectExitCode(0, termtest.OptExpectTimeout(time.Minute))

		if strings.Contains(cp.Snapshot(), search) {
			found = true
			break
		}
	}

	suite.Require().True(found, "Expecting to find %q", search)
}

func (suite *UpdateIntegrationTestSuite) installLatestReleaseVersion(ts *e2e.Session, dir string) {
	var cp *e2e.SpawnedCmd
	if runtime.GOOS != "windows" {
		oneLiner := fmt.Sprintf("sh <(curl -q https://platform.activestate.com/dl/cli/pdli01/install.sh) -f -n -t %s", dir)
		cp = ts.SpawnCmdWithOpts(
			"bash", e2e.OptArgs("-c", oneLiner),
		)
	} else {
		b, err := httputil.GetDirect("https://platform.activestate.com/dl/cli/pdli01/install.ps1")
		suite.Require().NoError(err)

		ps1File := filepath.Join(ts.Dirs.Work, "install.ps1")
		suite.Require().NoError(fileutils.WriteFile(ps1File, b))

		cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.OptArgs(ps1File, "-f", "-n", "-t", dir),
			e2e.OptAppendEnv("SHELL="),
		)
	}
	cp.Expect("Installation Complete", termtest.OptExpectTimeout(5*time.Minute))

	stateExec, err := installation.StateExecFromDir(dir)
	suite.NoError(err)

	suite.FileExists(stateExec)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateToCurrent() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	installDir := filepath.Join(ts.Dirs.Work, "install")
	err := fileutils.MkdirUnlessExists(installDir)
	suite.NoError(err)

	suite.installLatestReleaseVersion(ts, installDir)

	suite.testAutoUpdate(ts, installDir, e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.UpdateChannelEnvVarName, constants.ChannelName)))
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

	suite.testUpdate(ts, installDir, e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.UpdateChannelEnvVarName, constants.ChannelName)))
}
