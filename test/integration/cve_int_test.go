package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type CveIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CveIntegrationTestSuite) TestCve() {
	suite.T().Skip("This does not work right now") // DX-3252, CP-658
	suite.OnlyRunForTags(tagsuite.Cve)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("cve", "ActiveState-CLI/VulnerablePython-3.7")
	cp.Expect("Commit ID")
	cp.Expect("0b87e7a4-dc62-46fd-825b-9c35a53fe0a2")

	cp.Expect("Vulnerabilities")
	cp.Expect("CRITICAL")
	cp.Expect("Affected Packages")
	cp.Expect("tensorflow")
	cp.Expect("CRITICAL")
	cp.Expect("CVE-2019-16778")
	cp.ExpectExitCode(0)

	// make sure that we can select by commit id
	cp = ts.Spawn("cve", "ActiveState-CLI/VulnerablePython-3.7#3b222e23-64b9-4ca1-93ee-7b8a75b18c30")
	cp.Expect("Commit ID")
	cp.Expect("3b222e23-64b9-4ca1-93ee-7b8a75b18c30")

	cp.Expect("Vulnerabilities")
	cp.ExpectExitCode(0)
}

func (suite *CveIntegrationTestSuite) TestCveNoVulnerabilities() {
	// If you need to run this test comment the next line and provide a commit that has no CVE's
	suite.T().Skip("Skipping test because due to the nature of CVE's it's impossible to nail down a commit without CVE's.")
	suite.OnlyRunForTags(tagsuite.Cve)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareProject("ActiveState-CLI/small-python", "9733d11a-dfb3-41de-a37a-843b7c421db4")

	cp := ts.Spawn("cve")
	cp.Expect("No CVEs detected")
	cp.ExpectExitCode(0)
}

func (suite *CveIntegrationTestSuite) TestCveInvalidProject() {
	suite.OnlyRunForTags(tagsuite.Cve)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("cve", "invalid/invalid")
	cp.Expect("not found")

	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()
}

func (suite *CveIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Cve, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("cve", "-o", "editor")
	cp.Expect(`"project":`)
	cp.Expect(`"commitID":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestCveIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CveIntegrationTestSuite))
}
