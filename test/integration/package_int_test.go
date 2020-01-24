package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
)

type PackageIntegrationTestSuite struct {
	integration.Suite
}

func (suite *PackageIntegrationTestSuite) newSession(args ...string) (*integration.Session, func()) {
	dirs := suite.NewDirs()
	def := func() { dirs.Close() }
	defer func() { def() }()

	suite.PrepareActiveStateYAML(dirs.Work)

	ts := suite.NewSession(dirs, suite.ExecutablePath(), args...)
	cleanUp := func() { dirs.Close(); ts.Close() }
	def = func() {}

	return ts, cleanUp
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingSimple() {
	ts, cleanUp := suite.newSession("packages")
	defer cleanUp()

	ts.Expect("Name")
	ts.Expect("pytest")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listCommand() {
	ts, cleanUp := suite.newSession("packages", "list")
	defer cleanUp()

	ts.Expect("Name")
	ts.Expect("pytest")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitValid() {
	ts, cleanUp := suite.newSession(
		"packages", "--commit", "b350c879-b72a-48da-bbc2-d8d709a6182a",
	)
	defer cleanUp()

	ts.Expect("Name")
	ts.Expect("numpy")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitInvalid() {
	ts, cleanUp := suite.newSession("packages", "--commit", "junk")
	defer cleanUp()

	ts.Expect("Cannot obtain")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitUnknown() {
	ts, cleanUp := suite.newSession(
		"packages", "--commit", "00010001-0001-0001-0001-000100010001",
	)
	defer cleanUp()

	ts.Expect("No data")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitValidNoPackages() {
	ts, cleanUp := suite.newSession(
		"packages", "--commit", "cd674adb-e89a-48ff-95c6-ad52a177537b",
	)
	defer cleanUp()

	ts.Expect("No packages")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchSimple() {
	ts, cleanUp := suite.newSession("packages", "search", "request")
	defer cleanUp()

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
		ts.Expect(expectation)
	}
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithExactTerm() {
	ts, cleanUp := suite.newSession("packages", "search", "requests", "--exact-term")
	defer cleanUp()

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
		ts.Expect(expectation)
	}
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithExactTermWrongTerm() {
	ts, cleanUp := suite.newSession("packages", "search", "xxxrequestsxxx", "--exact-term")
	defer cleanUp()

	ts.Expect("No packages")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithLang() {
	ts, cleanUp := suite.newSession("packages", "search", "moose", "--language=perl")
	defer cleanUp()

	ts.Expect("Name")
	ts.Expect("MooseX-Getopt")
	ts.Expect("MooseX-Role-Parameterized")
	ts.Expect("MooseX-Role-WithOverloading")
	ts.Expect("MooX-Types-MooseLike")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithWrongLang() {
	ts, cleanUp := suite.newSession("packages", "search", "numpy", "--language=perl")
	defer cleanUp()

	ts.Expect("No packages")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithBadLang() {
	ts, cleanUp := suite.newSession("packages", "search", "numpy", "--language=bad")
	defer cleanUp()

	ts.Expect("Cannot obtain search")
	ts.Wait()
}

func (suite *PackageIntegrationTestSuite) PrepareActiveStateYAML(dir string) {
	asyData := `project: "https://platform.activestate.com/ActiveState-CLI/Python3"`
	suite.Suite.PrepareActiveStateYAML(dir, asyData)
}

func TestPackageIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PackageIntegrationTestSuite))
}
