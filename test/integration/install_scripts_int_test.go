package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"

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
		{"install-prbranch-with-version", constants.Version, constants.BranchName, "", ""},
		{"install-prbranch-and-activate", "", constants.BranchName, "ActiveState-CLI/small-python", ""},
		{"install-prbranch-and-activate-by-command", "", constants.BranchName, "", "ActiveState-CLI/small-python"},
	}

	for _, tt := range tests {
		suite.Run(fmt.Sprintf("%s (%s@%s)", tt.Name, tt.Version, tt.Channel), func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			suite.T().Setenv(constants.HomeEnvVarName, ts.Dirs.HomeDir)

			// Determine URL of install script.
			baseUrl := "https://state-tool.s3.amazonaws.com/update/state/"
			scriptBaseName := "install."
			if runtime.GOOS != "windows" {
				scriptBaseName += "sh"
			} else {
				scriptBaseName += "ps1"
			}
			scriptUrl := baseUrl + constants.BranchName + "/" + scriptBaseName

			// Fetch it.
			b, err := httputil.GetDirect(scriptUrl)
			suite.Require().NoError(err)
			script := filepath.Join(ts.Dirs.Work, scriptBaseName)
			suite.Require().NoError(fileutils.WriteFile(script, b))

			// Construct installer command to execute.
			argsPlain := []string{script}
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

			appInstallDir := filepath.Join(ts.Dirs.Work, "app")
			suite.NoError(fileutils.Mkdir(appInstallDir))

			var cp *e2e.SpawnedCmd
			if runtime.GOOS != "windows" {
				cp = ts.SpawnCmdWithOpts(
					"bash", e2e.OptArgs(argsWithActive...),
					e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
					e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
				)
			} else {
				cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.OptArgs(argsWithActive...),
					e2e.OptAppendEnv("SHELL="),
					e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
					e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
				)
			}

			expectStateToolInstallation(cp)

			if tt.Activate != "" || tt.ActivateByCommand != "" {
				cp.Expect("Creating a Virtual Environment")
				cp.Expect("Quick Start", termtest.OptExpectTimeout(time.Minute*2))
				// ensure that shell is functional
				cp.ExpectInput()

				cp.SendLine("python3 -c \"import sys; print(sys.copyright)\"")
				cp.Expect("ActiveState")
			}

			installPath, err := installation.InstallPathForBranch(constants.BranchName)
			suite.NoError(err)

			binPath := filepath.Join(installPath, "bin")

			statePath := filepath.Join(binPath, "state"+osutils.ExeExt)

			if runtime.GOOS == "windows" {
				installPath, err = osutils.BashifyPath(installPath)
				suite.NoError(err)

				statePath, err = osutils.BashifyPath(statePath)
				suite.NoError(err)
			}

			cp.SendLine("env | grep PATH")
			cp.Expect(installPath)
			cp.SendLine(statePath + " --version")
			cp.Expect("Version " + constants.Version)
			cp.Expect("Branch " + constants.BranchName)
			cp.Expect("Built")
			cp.SendLine("exit")

			cp.ExpectExitCode(0)

			stateExec, err := installation.StateExecFromDir(ts.Dirs.HomeDir)
			suite.NoError(err)
			suite.FileExists(stateExec)

			suite.assertBinDirContents(binPath)
			suite.assertCorrectVersion(ts, binPath, tt.Version, tt.Channel)
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
		})
	}
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstall_NonEmptyTarget() {
	suite.OnlyRunForTags(tagsuite.InstallScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work)
	argsPlain := []string{script, "-t", ts.Dirs.Work}
	argsPlain = append(argsPlain, "-b", constants.BranchName)
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
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstall_VersionDoesNotExist() {
	suite.OnlyRunForTags(tagsuite.InstallScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work)
	args := []string{script, "-t", ts.Dirs.Work}
	args = append(args, "-v", "does-not-exist")
	var cp *e2e.SpawnedCmd
	if runtime.GOOS != "windows" {
		cp = ts.SpawnCmdWithOpts("bash", e2e.OptArgs(args...))
	} else {
		cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.OptArgs(args...), e2e.OptAppendEnv("SHELL="))
	}
	cp.Expect("Could not download")
	cp.Expect("does-not-exist")
	cp.ExpectExitCode(1)
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

func expectStateToolInstallation(cp *e2e.SpawnedCmd) {
	cp.Expect("Preparing Installer for State Tool Package Manager")
	cp.Expect("Installation Complete", termtest.OptExpectTimeout(time.Minute))
}

// assertBinDirContents checks if given files are or are not in the bin directory
func (suite *InstallScriptsIntegrationTestSuite) assertBinDirContents(dir string) {
	fmt.Println("Searching dir:", dir)
	binFiles := listFilesOnly(dir)
	fmt.Println("Bin files:", binFiles)
	suite.Contains(binFiles, "state"+osutils.ExeExt)
	suite.Contains(binFiles, "state-svc"+osutils.ExeExt)
}

// listFilesOnly is a helper function for assertBinDirContents filtering a directory recursively for base filenames
// It allows for simple and coarse checks if a file exists or does not exist in the directory structure
func listFilesOnly(dir string) []string {
	files := fileutils.ListDirSimple(dir, true)
	files = funk.Filter(files, func(f string) bool {
		return !fileutils.IsDir(f)
	}).([]string)
	return funk.Map(files, filepath.Base).([]string)
}

func (suite *InstallScriptsIntegrationTestSuite) assertCorrectVersion(ts *e2e.Session, installDir, expectedVersion, expectedBranch string) {
	type versionData struct {
		Version string `json:"version"`
		Branch  string `json:"branch"`
	}

	stateExec, err := installation.StateExecFromDir(installDir)
	suite.NoError(err)

	cp := ts.SpawnCmd(stateExec, "--version", "--output=json")
	cp.ExpectExitCode(0)
	actual := versionData{}
	out := cp.StrippedSnapshot()
	fmt.Println("Stripped snapshot:", out)
	suite.Require().NoError(json.Unmarshal([]byte(out), &actual))

	if expectedVersion != "" {
		suite.Equal(expectedVersion, actual.Version)
	}
	if expectedBranch != "" {
		suite.Equal(expectedBranch, actual.Branch)
	}
}

func TestInstallScriptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallScriptsIntegrationTestSuite))
}
