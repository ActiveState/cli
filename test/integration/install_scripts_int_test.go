package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"

	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type InstallScriptsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstall() {
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	tests := []struct {
		Name              string
		Version           string
		Channel           string
		Activate          string
		ActivateByCommand string
	}{
		// {"install-release-latest", "", "release", "", ""},
		{"install-prbranch", "", "", "", ""},
		{"install-prbranch-with-version", constants.Version, constants.ChannelName, "", ""},
		{"install-prbranch-and-activate", "", constants.ChannelName, "ActiveState-CLI/small-python", ""},
		{"install-prbranch-and-activate-by-command", "", constants.ChannelName, "", "ActiveState-CLI/small-python"},
	}

	for _, tt := range tests {
		suite.Run(fmt.Sprintf("%s (%s@%s)", tt.Name, tt.Version, tt.Channel), func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			// Determine URL of install script.
			baseUrl := "https://state-tool.s3.amazonaws.com/update/state/"
			scriptBaseName := "install."
			if runtime.GOOS != "windows" {
				scriptBaseName += "sh"
			} else {
				scriptBaseName += "ps1"
			}
			scriptUrl := baseUrl + constants.ChannelName + "/" + scriptBaseName

			// Fetch it.
			b, err := httputil.GetDirect(scriptUrl)
			suite.Require().NoError(err)
			script := filepath.Join(ts.Dirs.Work, scriptBaseName)
			suite.Require().NoError(fileutils.WriteFile(script, b))

			// Construct installer command to execute.
			installDir := filepath.Join(ts.Dirs.Work, "install")
			argsPlain := []string{script}
			argsPlain = append(argsPlain, "-t", installDir)
			argsPlain = append(argsPlain, "-n")
			if tt.Channel != "" {
				argsPlain = append(argsPlain, "-b", tt.Channel)
			}
			if tt.Version != "" {
				argsPlain = append(argsPlain, "-v", tt.Version)
			}

			argsWithActive := append(argsPlain, "-f")
			if tt.Activate != "" {
				argsWithActive = append(argsWithActive, "--activate", tt.Activate)
			}
			if tt.ActivateByCommand != "" {
				cmd := fmt.Sprintf("state activate %s", tt.ActivateByCommand)
				if runtime.GOOS == "windows" {
					cmd = "'" + cmd + "'"
				}
				argsWithActive = append(argsWithActive, "-c", cmd)
			}

			// Make the directory to install to.
			appInstallDir := filepath.Join(ts.Dirs.Work, "app")
			suite.NoError(fileutils.Mkdir(appInstallDir))

			// Perform the installation.
			cmd := "bash"
			opts := []e2e.SpawnOptSetter{
				e2e.OptArgs(argsWithActive...),
				e2e.OptAppendEnv(constants.DisableRuntime + "=false"),
				e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
				e2e.OptAppendEnv(fmt.Sprintf("%s=FOO", constants.OverrideSessionTokenEnvVarName)),
				e2e.OptAppendEnv(fmt.Sprintf("%s=false", constants.DisableActivateEventsEnvVarName)),
			}
			if runtime.GOOS == "windows" {
				cmd = "powershell.exe"
				opts = append(opts, e2e.OptAppendEnv("SHELL="))
			}
			cp := ts.SpawnCmdWithOpts(cmd, opts...)
			cp.Expect("Preparing Installer for State Tool Package Manager")
			cp.Expect("Installation Complete", e2e.RuntimeSourcingTimeoutOpt)

			if tt.Activate != "" || tt.ActivateByCommand != "" {
				cp.Expect("Creating a Virtual Environment")
				cp.Expect("Quick Start", e2e.RuntimeSourcingTimeoutOpt)
				// ensure that shell is functional
				cp.ExpectInput()

				cp.SendLine("python3 -c \"import sys; print(sys.copyright)\"")
				cp.Expect("ActiveState")
			}

			cp.SendLine("state --version")
			cp.Expect("Version " + constants.Version)
			cp.Expect("Channel " + constants.ChannelName)
			cp.Expect("Built")
			cp.SendLine("exit")

			cp.ExpectExitCode(0)

			stateExec, err := installation.StateExecFromDir(installDir)
			suite.NoError(err)
			suite.FileExists(stateExec)

			suite.assertBinDirContents(filepath.Join(installDir, "bin"))
			suite.assertCorrectVersion(ts, installDir, tt.Version, tt.Channel)
			suite.assertAnalytics(ts)
			suite.DirExists(ts.Dirs.Config)

			// Verify that can install overtop
			if runtime.GOOS != "windows" {
				cp = ts.SpawnCmdWithOpts("bash", e2e.OptArgs(argsPlain...))
			} else {
				cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.OptArgs(argsPlain...),
					e2e.OptAppendEnv("SHELL="),
				)
			}
			cp.Expect("successfully installed")
			cp.ExpectInput()
			cp.SendLine("exit")
			cp.ExpectExitCode(0)
			if runtime.GOOS == "windows" {
				ts.IgnoreLogErrors() // Follow-up DX-2678
			}
		})
	}
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstall_NonEmptyTarget() {
	suite.OnlyRunForTags(tagsuite.InstallScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work)
	argsPlain := []string{script, "-t", ts.Dirs.Work, "-n"}
	argsPlain = append(argsPlain, "-b", constants.ChannelName)
	var cp *e2e.SpawnedCmd
	if runtime.GOOS != "windows" {
		cp = ts.SpawnCmdWithOpts("bash", e2e.OptArgs(argsPlain...))
	} else {
		cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.OptArgs(argsPlain...), e2e.OptAppendEnv("SHELL="))
	}
	cp.Expect("Installation path must be an empty directory")

	// Originally this was ExpectExitCode(1), but particularly on Windows this turned out to be unreliable. Probably
	// because of powershell.
	// Since we asserted the expected error above we don't need to go on a wild goose chase here.
	cp.ExpectExit()
	ts.IgnoreLogErrors()
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstall_VersionDoesNotExist() {
	suite.OnlyRunForTags(tagsuite.InstallScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work)
	args := []string{script, "-t", ts.Dirs.Work, "-n"}
	args = append(args, "-v", "does-not-exist")
	var cp *e2e.SpawnedCmd
	if runtime.GOOS != "windows" {
		cp = ts.SpawnCmdWithOpts("bash", e2e.OptArgs(args...))
	} else {
		cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.OptArgs(args...), e2e.OptAppendEnv("SHELL="))
	}
	if !condition.OnCI() || runtime.GOOS == "windows" {
		// For some reason on Linux and macOS, there is no terminal output on CI. It works locally though.
		cp.Expect("Could not download")
	}
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

// scriptPath returns the path to an installation script copied to targetDir, if useTestUrl is true, the install script is modified to download from the local test server instead
func scriptPath(t *testing.T, targetDir string) string {
	ext := ".ps1"
	if runtime.GOOS != "windows" {
		ext = ".sh"
	}
	name := "install" + ext
	root := environment.GetRootPathUnsafe()
	subdir := "installers"

	source := filepath.Join(root, subdir, name)
	if !fileutils.FileExists(source) {
		t.Fatalf("Could not find install script %s", source)
	}

	target := filepath.Join(targetDir, filepath.Base(source))
	err := fileutils.CopyFile(source, target)
	require.NoError(t, err)

	return target
}

// assertBinDirContents checks if given files are or are not in the bin directory
func (suite *InstallScriptsIntegrationTestSuite) assertBinDirContents(dir string) {
	binFiles := suite.listFilesOnly(dir)
	suite.Contains(binFiles, "state"+osutils.ExeExtension)
	suite.Contains(binFiles, "state-svc"+osutils.ExeExtension)
}

// listFilesOnly is a helper function for assertBinDirContents filtering a directory recursively for base filenames
// It allows for simple and coarse checks if a file exists or does not exist in the directory structure
func (suite *InstallScriptsIntegrationTestSuite) listFilesOnly(dir string) []string {
	files, err := fileutils.ListDirSimple(dir, true)
	suite.Require().NoError(err)
	files = funk.Filter(files, func(f string) bool {
		return !fileutils.IsDir(f)
	}).([]string)
	return funk.Map(files, filepath.Base).([]string)
}

func (suite *InstallScriptsIntegrationTestSuite) assertCorrectVersion(ts *e2e.Session, installDir, expectedVersion, expectedChannel string) {
	type versionData struct {
		Version string `json:"version"`
		Channel string `json:"channel"`
	}

	stateExec, err := installation.StateExecFromDir(installDir)
	suite.NoError(err)

	cp := ts.SpawnCmd(stateExec, "--version", "--output=json")
	cp.ExpectExitCode(0)
	actual := versionData{}
	out := cp.StrippedSnapshot()
	suite.Require().NoError(json.Unmarshal([]byte(out), &actual))

	if expectedVersion != "" {
		suite.Equal(expectedVersion, actual.Version)
	}
	if expectedChannel != "" {
		suite.Equal(expectedChannel, actual.Channel)
	}
}

func (suite *InstallScriptsIntegrationTestSuite) assertAnalytics(ts *e2e.Session) {
	// Verify analytics reported a non-empty sessionToken.
	sessionTokenFound := false
	events := parseAnalyticsEvents(suite, ts)
	suite.Require().NotEmpty(events)
	for _, event := range events {
		if event.Category == anaConst.CatInstallerFunnel && event.Dimensions != nil {
			suite.Assert().NotEmpty(*event.Dimensions.SessionToken)
			sessionTokenFound = true
			break
		}
	}
	suite.Assert().True(sessionTokenFound, "sessionToken was not found in analytics")
}

func TestInstallScriptsIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InstallScriptsIntegrationTestSuite))
}
