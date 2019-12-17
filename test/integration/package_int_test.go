package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
)

type PackageIntegrationTestSuite struct {
	integration.Suite
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingSimple() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("package")
	suite.Expect("Name")
	suite.Expect("pytest")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitValid() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("package", "--commit", "b350c879-b72a-48da-bbc2-d8d709a6182a")
	suite.Expect("Name")
	suite.Expect("numpy")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitInvalid() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("package", "--commit", "junk")
	suite.Expect("Cannot obtain")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitUnknown() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("package", "--commit", "00010001-0001-0001-0001-000100010001")
	suite.Expect("No data")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) PrepareActiveStateYAML(dir string) {
	asyData := `project: "https://platform.activestate.com/ActiveState-CLI/Python3"`
	suite.Suite.PrepareActiveStateYAML(dir, asyData)
}

func TestPackageIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(PackageIntegrationTestSuite))
}
