package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
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
		// {"install-release-latest", "", constants.ReleaseChannel},
		// {"install-prbranch", "", ""},
		// {"install-prbranch-with-version", constants.Version, constants.ChannelName},
		{"install-prbranch-and-channel", "", constants.ChannelName},
	}

	for _, tt := range tests {
		suite.Run(fmt.Sprintf("%s (%s@%s)", tt.Name, tt.Version, tt.Channel), func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			suite.setupTest(ts)

			installPath := filepath.Join(ts.Dirs.Work, "install")
			stateExePath := filepath.Join(installPath, "bin", constants.StateCmd+osutils.ExeExtension)

			args := []string{"-n"}
			if tt.Version != "" {
				args = append(args, "--version", tt.Version)
			}
			if tt.Channel != "" {
				args = append(args, "--channel", tt.Channel)
			}

			appInstallDir := filepath.Join(ts.Dirs.Work, "app")
			suite.NoError(fileutils.Mkdir(appInstallDir))

			cp := ts.SpawnCmdWithOpts(
				suite.remoteInstallerExe,
				e2e.OptArgs(args...),
				e2e.OptAppendEnv(constants.InstallPathOverrideEnvVarName+"="+installPath),
				e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
			)

			cp.Expect("Terms of Service")
			cp.SendLine("Y")
			cp.Expect("Installing")
			cp.Expect("Installation Complete")
			cp.Expect("Press ENTER to exit")
			cp.SendEnter()
			cp.ExpectExitCode(0)

			suite.Require().FileExists(stateExePath)

			cp = ts.SpawnCmdWithOpts(
				stateExePath,
				e2e.OptArgs("--version"),
				e2e.OptAppendEnv(constants.InstallPathOverrideEnvVarName+"="+installPath),
			)
			if tt.Version != "" {
				cp.Expect("Version " + tt.Version)
			}
			if tt.Channel != "" {
				cp.Expect("Channel " + tt.Channel)
			}
			cp.Expect("Built")
			cp.ExpectExitCode(0)

			// Verify analytics reported the correct sessionToken.
			sessionTokenFound := false
			events := parseAnalyticsEvents(suite, ts)
			suite.Require().NotEmpty(events)
			for _, event := range events {
				if event.Category == anaConst.CatUpdates && event.Dimensions != nil {
					suite.Assert().Contains(*event.Dimensions.SessionToken, constants.RemoteInstallerVersion)
					sessionTokenFound = true
					break
				}
			}
			suite.Assert().True(sessionTokenFound, "sessionToken was not found in analytics")
		})
	}
}

func (s *RemoteInstallIntegrationTestSuite) setupTest(ts *e2e.Session) {
	root := environment.GetRootPathUnsafe()
	buildDir := fileutils.Join(root, "build")
	installerExe := filepath.Join(buildDir, constants.StateRemoteInstallerCmd+osutils.ExeExtension)
	if !fileutils.FileExists(installerExe) {
		s.T().Fatal("E2E tests require a state-remote-installer binary. Run `state run build-installer`.")
	}
	s.remoteInstallerExe = ts.CopyExeToDir(installerExe, filepath.Join(ts.Dirs.Base, "installer"))
}

func TestRemoteInstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RemoteInstallIntegrationTestSuite))
}
