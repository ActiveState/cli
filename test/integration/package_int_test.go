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
	suite.Expect("python")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitValid() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("package", "--commit", "c780f643-724b-49bb-aca9-194e3c072f64")
	suite.Expect("Name")
	suite.Expect("python")
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
