package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal-as/constants"
	"github.com/ActiveState/cli/internal-as/environment"
	"github.com/ActiveState/cli/internal-as/exeutils"
	"github.com/ActiveState/cli/internal-as/fileutils"
	"github.com/ActiveState/cli/internal-as/osutils"
	"github.com/ActiveState/cli/internal-as/testhelpers/e2e"
	"github.com/ActiveState/cli/internal-as/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type RemoteInstallIntegrationTestSuite struct {
	tagsuite.Suite
	remoteInstallerExe string
}

func (suite *RemoteInstallIntegrationTestSuite) TestInstall() {
	suite.OnlyRunForTags(tagsuite.RemoteInstaller, tagsuite.Critical)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	tests := []struct {
		Name    string
		Version string
		Channel string
	}{
		// Disabled until the target installers support the installpath override env var: DX-1350
		// {"install-release-latest", "", constants.ReleaseBranch},
		// {"install-prbranch", "", ""},
		// {"install-prbranch-with-version", constants.Version, constants.BranchName},
		{"install-prbranch-and-branch", "", constants.BranchName},
	}

	for _, tt := range tests {
		suite.Run(fmt.Sprintf("%s (%s@%s)", tt.Name, tt.Version, tt.Channel), func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			suite.setupTest(ts)

			installPath := filepath.Join(ts.Dirs.Work, "install")
			stateExePath := filepath.Join(installPath, "bin", constants.StateCmd+exeutils.Extension)

			args := []string{}
			if tt.Version != "" {
				args = append(args, "--version", tt.Version)
			}
			if tt.Channel != "" {
				args = append(args, "--channel", tt.Channel)
			}

			cp := ts.SpawnCmdWithOpts(
				suite.remoteInstallerExe,
				e2e.WithArgs(args...),
				e2e.AppendEnv(constants.InstallPathOverrideEnvVarName+"="+installPath),
			)

			cp.Expect("Terms of Service")
			cp.SendLine("y")
			cp.Expect("Installing")
			cp.Expect("Installation Complete")
			cp.Expect("Press ENTER to exit")
			cp.SendLine("")
			cp.ExpectExitCode(0)

			suite.Require().FileExists(stateExePath)

			cp = ts.SpawnCmdWithOpts(
				stateExePath,
				e2e.WithArgs("--version"),
				e2e.AppendEnv(constants.InstallPathOverrideEnvVarName+"="+installPath),
			)
			if tt.Version != "" {
				cp.Expect("Version " + tt.Version)
			}
			if tt.Channel != "" {
				cp.Expect("Branch " + tt.Channel)
			}
			cp.Expect("Built")
			cp.ExpectExitCode(0)
		})
	}
}

func (s *RemoteInstallIntegrationTestSuite) setupTest(ts *e2e.Session) {
	root := environment.GetRootPathUnsafe()
	buildDir := fileutils.Join(root, "build")
	installerExe := filepath.Join(buildDir, constants.StateRemoteInstallerCmd+osutils.ExeExt)
	if !fileutils.FileExists(installerExe) {
		s.T().Fatal("E2E tests require a state-remote-installer binary. Run `state run build-installer`.")
	}
	s.remoteInstallerExe = ts.CopyExeToDir(installerExe, filepath.Join(ts.Dirs.Base, "installer"))
}

func TestRemoteInstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RemoteInstallIntegrationTestSuite))
}
