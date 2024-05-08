package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type InfoIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InfoIntegrationTestSuite) TestInfo_LatestVersion() {
	suite.OnlyRunForTags(tagsuite.Info)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint", "--language", "python")
	cp.Expect("Package Information")
	cp.Expect("Author")
	cp.Expect("Version(s) Available")
	cp.ExpectExitCode(0)
}

func (suite *InfoIntegrationTestSuite) TestInfo_SpecificVersion() {
	suite.OnlyRunForTags(tagsuite.Info)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint@0.28.0", "--language", "python")
	cp.Expect("Package Information: pylint@0.28.0")
	cp.Expect("Author")
	cp.Expect("Logilab")
	cp.ExpectExitCode(0)
}

func (suite *InfoIntegrationTestSuite) TestInfo_UnavailableVersion() {
	suite.OnlyRunForTags(tagsuite.Info)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint@9.9.9", "--language", "python")
	cp.Expect("Could not find version 9.9.9 for package pylint")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *InfoIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Info, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint", "--language", "python", "-o", "json")
	cp.Expect(`"description":`)
	cp.Expect(`"authors":`)
	cp.Expect(`"version":`)
	cp.ExpectExitCode(0)
	//AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("info", "pylint@9.9.9", "--language", "python", "--output", "editor")
	cp.Expect(`"error":`)
	cp.ExpectExitCode(1)
	AssertValidJSON(suite.T(), cp)
	ts.IgnoreLogErrors()
}

func TestInfoIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InfoIntegrationTestSuite))
}
