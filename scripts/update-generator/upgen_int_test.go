package main_test

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

	suite.Wait()
	suite.Equal(0, suite.ExitCode(), "Exits with code 0")

	suite.SpawnCustom(filepath.Join(tempPath, platform+exe), "--version")
	suite.Expect(constants.BuildNumber)
	suite.Wait()
}

func TestUpdateGenIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	suite.Run(t, new(UpdateGenIntegrationTestSuite))

	// parallel doesn't work with these due to contamination. The RunParallel function does not seem to allow for
	// setting up individual tests
	//expect.RunParallel(t, new(UpdateGenIntegrationTestSuite))
}
