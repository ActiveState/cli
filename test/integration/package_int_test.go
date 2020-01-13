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

func (suite *PackageIntegrationTestSuite) TestPackage_listCommand() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "list")
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

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitValidNoPackages() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "--commit", "cd674adb-e89a-48ff-95c6-ad52a177537b")
	suite.Expect("No packages")
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchSimple() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "search", "request")
	expectations := []string{
		"Name",
		"aws-requests-auth",
		"django-request-logging",
		"requests",
		"2.10.0",
		"2.18.4",
		"2.21.0",
		"2.22.0",
		"2.3",
		"requests-oauthlib",
		"requests3",
		"requests_gpgauthlib",
		"requestsexceptions",
		"robotframework-requests",
	}
	for _, expectation := range expectations {
		suite.Expect(expectation)
	}
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithExactTerm() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "search", "requests", "--exact-term")
	expectations := []string{
		"Name",
		"requests",
		"2.10.0",
		"2.18.4",
		"2.21.0",
		"2.22.0",
		"2.3",
		"---",
	}
	for _, expectation := range expectations {
		suite.Expect(expectation)
	}
	suite.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithExactTermWrongTerm() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("packages", "search", "xxxrequestsxxx", "--exact-term")
	suite.Expect("No packages")
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
	suite.Run(t, new(PackageIntegrationTestSuite))
}
