package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
)

type UpdateGenIntegrationTestSuite struct {
	integration.Suite
}

func (suite *UpdateGenIntegrationTestSuite) TestUpdateBits() {
	root := environment.GetRootPathUnsafe()

	ext := ".tar.gz"
	exe := ""
	if runtime.GOOS == "windows" {
		ext = ".zip"
		exe = ".exe"
	}
	platform := runtime.GOOS + "-" + runtime.GOARCH

	archivePath := filepath.Join(root, "public/update", constants.BranchName, constants.Version, platform+ext)
	suite.Require().FileExists(archivePath)

	tempPath, err := ioutil.TempDir("", "")
	suite.Require().NoError(err)

	if runtime.GOOS == "windows" {
		suite.SpawnCustom("powershell.exe", "-nologo", "-noprofile", "-command",
			fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s'", archivePath, tempPath))
	} else {
		suite.SpawnCustom("tar", "-C", tempPath, "-xf", archivePath)
	}

	ps, err := suite.Wait()
	suite.Require().NoError(err)
	suite.Equal(0, ps.ExitCode(), "Exits with code 0")

	suite.SpawnCustom(filepath.Join(tempPath, platform+exe), "--version")
	suite.Expect(constants.BuildNumber)
	suite.Wait()
}

func TestUpdateGenIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateGenIntegrationTestSuite))
}
