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

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState/ActivePython-3.7`)

	cp := ts.Spawn("cve")
	cp.Expect("ActivePython-3.7")
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

	cp := ts.Spawn("cve", "report", "ActiveState/ActivePython-3.7")
	cp.Expect("Commit ID")

	cp.Expect("Vulnerabilities")
	cp.Expect("6")
	cp.Expect("CRITICAL")
	cp.Expect("13 Affected Packages")
	cp.Expect("tensorflow")
	cp.Expect("1.12.0")
	cp.Expect("CRITICAL")
	cp.Expect("CVE-2019-16778")
	cp.ExpectExitCode(0)
}

func (suite *CveIntegrationTestSuite) TestCveNoVulnerabilities() {
	suite.OnlyRunForTags(tagsuite.Cve)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/small-python`)

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

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/invalid/invalid`)

	cp := ts.Spawn("cve")
	cp.Expect("Found no project with specified organization and name")

	cp.ExpectNotExitCode(0)
}

func TestCveIntegraionTestSuite(t *testing.T) {
	suite.Run(t, new(CveIntegrationTestSuite))
}
