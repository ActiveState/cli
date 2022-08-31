package integration

import (
	"path/filepath"
	"testing"

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
	installerExe string
}

func (suite *RemoteInstallIntegrationTestSuite) TestInstall() {
	suite.OnlyRunForTags(tagsuite.RemoteInstaller, tagsuite.Critical)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.setupTest(ts)

	cp := ts.SpawnCmdWithOpts(suite.installerExe, e2e.WithArgs("--channel", constants.ReleaseBranch))
	cp.Expect("Terms of Service")
	cp.SendLine("y")
	cp.Expect("Installing")
	cp.Expect("Installation complete. Press enter to exit")
	cp.SendLine("")
	cp.ExpectExitCode(0)
}

func (s *RemoteInstallIntegrationTestSuite) setupTest(ts *e2e.Session) {
	root := environment.GetRootPathUnsafe()
	buildDir := fileutils.Join(root, "build")
	installerExe := filepath.Join(buildDir, constants.StateRemoteInstallerCmd+osutils.ExeExt)
	if !fileutils.FileExists(installerExe) {
		s.T().Fatal("E2E tests require a state-remote-installer binary. Run `state run build-installer`.")
	}
	s.installerExe = ts.CopyExeToDir(installerExe, filepath.Join(ts.Dirs.Base, "installer"))
}

func TestRemoteInstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RemoteInstallIntegrationTestSuite))
}
