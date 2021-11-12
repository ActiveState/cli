package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type CveIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CveIntegrationTestSuite) TestCveSummary() {
	suite.OnlyRunForTags(tagsuite.Cve)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/VulnerablePython-3.7?commitID=0b87e7a4-dc62-46fd-825b-9c35a53fe0a2`)

	cp := ts.Spawn("cve")
	cp.Expect("VulnerablePython-3.7")
	cp.Expect("0b87e7a4-dc62-46fd-825b-9c35a53fe0a2")

	cp.Expect("Vulnerabilities")
	cp.Expect("6")
	cp.Expect("CRITICAL")
	cp.Expect("13 Affected Packages")
	cp.Expect("tensorflow")
	cp.Expect("1.12.0")
	cp.Expect("18")
	cp.ExpectExitCode(0)
}

func (suite *CveIntegrationTestSuite) TestCveReport() {
	suite.OnlyRunForTags(tagsuite.Cve)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("cve", "report", "ActiveState-CLI/VulnerablePython-3.7")
	cp.Expect("Commit ID")
	cp.Expect("0b87e7a4-dc62-46fd-825b-9c35a53fe0a2")

	cp.Expect("Vulnerabilities")
	cp.Expect("6")
	cp.Expect("CRITICAL")
	cp.Expect("13 Affected Packages")
	cp.Expect("tensorflow")
	cp.Expect("1.12.0")
	cp.Expect("CRITICAL")
	cp.Expect("CVE-2019-16778")
	cp.ExpectExitCode(0)

	// make sure that we can select by commit id
	cp = ts.Spawn("cve", "report", "ActiveState-CLI/VulnerablePython-3.7#3b222e23-64b9-4ca1-93ee-7b8a75b18c30")
	cp.Expect("Commit ID")
	cp.Expect("3b222e23-64b9-4ca1-93ee-7b8a75b18c30")

	cp.Expect("Vulnerabilities")
	cp.Expect("6")
	cp.ExpectExitCode(0)
}

func (suite *CveIntegrationTestSuite) TestCveNoVulnerabilities() {
	suite.OnlyRunForTags(tagsuite.Cve)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/small-python?commitID=9733d11a-dfb3-41de-a37a-843b7c421db4`)

	cp := ts.Spawn("cve")
	cp.Expect("No CVEs detected")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("cve", "report")
	cp.Expect("No CVEs detected")
	cp.ExpectExitCode(0)
}

func (suite *CveIntegrationTestSuite) TestCveInvalidProject() {
	suite.OnlyRunForTags(tagsuite.Cve)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("cve", "report", "invalid/invalid")
	cp.Expect("Found no project with specified organization and name")

	cp.ExpectNotExitCode(0)
}

func TestCveIntegraionTestSuite(t *testing.T) {
	suite.Run(t, new(CveIntegrationTestSuite))
}
