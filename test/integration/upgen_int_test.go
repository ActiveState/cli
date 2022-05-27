package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
)

type UpdateGenIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *UpdateGenIntegrationTestSuite) TestUpdateBits() {
	suite.OnlyRunForTags(tagsuite.CLIDeploy, tagsuite.Critical)
	root := environment.GetRootPathUnsafe()

	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	hostArch := runtime.GOARCH
	if runtime.GOOS == "darwin" && hostArch == "arm64" {
		hostArch = "amd64"
	}
	platform := runtime.GOOS + "-" + hostArch

	archivePath := filepath.Join(root, "build/update", constants.BranchName, constants.Version, platform, fmt.Sprintf("state-%s-%s%s", platform, constants.Version, ext))
	suite.Require().FileExists(archivePath, "Make sure you ran 'state run generate-update'")
	suite.T().Logf("file %s exists\n", archivePath)

	tempPath, err := ioutil.TempDir("", "")
	suite.Require().NoError(err)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	var cp *termtest.ConsoleProcess

	if runtime.GOOS == "windows" {
		cp = ts.SpawnCmd("powershell.exe", "-nologo", "-noprofile", "-command",
			fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s'", archivePath, tempPath))
	} else {
		cp = ts.SpawnCmd("tar", "-C", tempPath, "-xf", archivePath)
	}

	cp.ExpectExitCode(0)

	baseDir := filepath.Join(tempPath, constants.ToplevelInstallArchiveDir)
	stateExec := filepath.Join(baseDir, installation.BinDirName, constants.StateCmd+osutils.ExeExt)

	cp = ts.SpawnCmd(stateExec, "--version")
	cp.Expect(constants.RevisionHashShort)
	cp.ExpectExitCode(0)
}

func TestUpdateGenIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateGenIntegrationTestSuite))
}
