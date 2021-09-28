package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
)

type InstallScriptsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstall() {
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	tests := []struct {
		Name     string
		Version  string
		Channel  string
		Activate string
	}{
		// {"install-release-latest", "", "release", ""},
		{"install-prbranch", "", constants.BranchName, ""},
		{"install-prbranch-with-version", constants.Version, constants.BranchName, ""},
		{"install-prbranch-and-activate", "", constants.BranchName, "ActiveState-CLI/small-python"},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			dir, err := installation.InstallPath()
			suite.Require().NoError(err)
			var extraEnv []string
			if runtime.GOOS == "linux" {
				dir, err = ioutil.TempDir("", "temp_home*")
				suite.Require().NoError(err)
				extraEnv = append(extraEnv, fmt.Sprintf("HOME=%s", dir), fmt.Sprintf("_TEST_SYSTEM_PATH=%s", dir))
			}

			ts := e2e.New(suite.T(), false, extraEnv...)
			defer ts.Close()

			script := scriptPath(suite.T(), ts.Dirs.Work)
			args := []string{script, "-t", ts.Dirs.Work, "-b", tt.Channel, "-v", tt.Version}

			if tt.Activate != "" {
				args = append(args, "--activate", tt.Activate)
			}

			var cp *termtest.ConsoleProcess
			if runtime.GOOS != "windows" {
				cp = ts.SpawnCmdWithOpts(
					"bash", e2e.WithArgs(args...),
					e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
				)
			} else {
				cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.WithArgs(args...),
					e2e.AppendEnv("SHELL="),
					e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
				)
			}

			expectStateToolInstallation(cp)

			if tt.Activate != "" {
				cp.Expect("Creating a Virtual Environment")
				cp.Expect("Quick Start")
				// ensure that shell is functional
				cp.WaitForInput()

				cp.SendLine("python3 -c \"import sys; print(sys.copyright)\"")
				cp.Expect("ActiveState Software Inc.")

				cp.SendLine("exit")
			}

			cp.ExpectExitCode(0)

			state := appinfo.StateApp(ts.Dirs.Work)
			suite.FileExists(state.Exec())

			suite.assertBinDirContents(filepath.Join(ts.Dirs.Work, "bin"))
			suite.assertCorrectVersion(ts, tt.Version, tt.Channel)
			suite.DirExists(ts.Dirs.Config)
		})
	}
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


func expectStateToolInstallation(cp *termtest.ConsoleProcess) {
	cp.Expect("Preparing Installer for State Tool Package Manager")
	cp.Expect("Installation Complete", time.Second*20)
}

// assertBinDirContents checks if given files are or are not in the bin directory
func (suite *InstallScriptsIntegrationTestSuite) assertBinDirContents(dir string) {
	binFiles := listFilesOnly(dir)
	suite.Contains(binFiles, "state"+osutils.ExeExt)
	suite.Contains(binFiles, "state-tray"+osutils.ExeExt)
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

func (suite *InstallScriptsIntegrationTestSuite) assertCorrectVersion(ts *e2e.Session, expectedVersion, expectedBranch string) {
	type versionData struct {
		Version string `json:"version"`
		Branch  string `json:"branch"`
	}

	state := appinfo.StateApp(ts.Dirs.Work)
	cp := ts.SpawnCmd(state.Exec(), "--version", "--output=json")
	cp.ExpectExitCode(0)
	actual := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &actual)

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
