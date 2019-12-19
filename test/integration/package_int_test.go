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

	suite.Spawn("packages")
	suite.Expect("Name")
	suite.Expect("pytest")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitValid() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "--commit", "b350c879-b72a-48da-bbc2-d8d709a6182a")
	suite.Expect("Name")
	suite.Expect("numpy")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitInvalid() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "--commit", "junk")
	suite.Expect("Cannot obtain")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitUnknown() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "--commit", "00010001-0001-0001-0001-000100010001")
	suite.Expect("No data")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchSimple() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "search", "numpy")
	suite.Expect("Name")
	suite.Expect("msgpack-numpy")
	suite.Expect("numpy")
	suite.Expect("1.14.3")
	suite.Expect("1.16.1")
	suite.Expect("1.16.2")
	suite.Expect("numpy-stl")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithLang() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "search", "moose", "--language=perl")
	suite.Expect("Name")
	suite.Expect("MooseX-Getopt")
	suite.Expect("MooseX-Role-Parameterized")
	suite.Expect("MooseX-Role-WithOverloading")
	suite.Expect("MooX-Types-MooseLike")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithWrongLang() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "search", "numpy", "--language=perl")
	suite.Expect("No packages")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithBadLang() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "search", "numpy", "--language=bad")
	suite.Expect("Cannot obtain search")
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
