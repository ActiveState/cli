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

	archivePath := filepath.Join(root, "build/update", constants.BranchName, constants.VersionNumber, platform, fmt.Sprintf("state-%s-%s%s", platform, constants.Version, ext))
	suite.Require().FileExists(archivePath, "Make sure you ran 'state run generate-update'")
	suite.T().Logf("file %s exists\n", archivePath)

	tempPath, err := ioutil.TempDir("", "")
	suite.Require().NoError(err)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	var cp *e2e.SpawnedCmd

	if runtime.GOOS == "windows" {
		cp = ts.SpawnCmd("powershell.exe", "-nologo", "-noprofile", "-command",
			fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s'", archivePath, tempPath))
	} else {
		cp = ts.SpawnCmd("tar", "-C", tempPath, "-xf", archivePath)
	}

	cp.ExpectExitCode(0)

	baseDir := filepath.Join(tempPath, constants.ToplevelInstallArchiveDir)
	suite.FileExists(filepath.Join(baseDir, installation.BinDirName, constants.StateCmd+osutils.ExeExtension))
	suite.FileExists(filepath.Join(baseDir, installation.BinDirName, constants.StateSvcCmd+osutils.ExeExtension))
}

func TestUpdateGenIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateGenIntegrationTestSuite))
}
