package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
	"github.com/stretchr/testify/suite"
)

type InfoIntegrationTestSuite struct {
	testsuite.Suite
}

func (suite *InfoIntegrationTestSuite) TestInfo_LatestVersion() {
	suite.OnlyRunForTags(testsuite.TagInfo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint", "--language", "python")
	cp.Expect("Details for version")
	cp.Expect("Authors")
	cp.Expect("Python Code Quality Authority")
	cp.Expect("Version(s) Available")
	cp.ExpectExitCode(0)
}

func (suite *InfoIntegrationTestSuite) TestInfo_SpecificVersion() {
	suite.OnlyRunForTags(testsuite.TagInfo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint@0.28.0", "--language", "python")
	cp.Expect("Details for version 0.28.0")
	cp.Expect("Authors")
	cp.Expect("Logilab")
	cp.ExpectExitCode(0)
}

func (suite *InfoIntegrationTestSuite) TestInfo_UnavailableVersion() {
	suite.OnlyRunForTags(testsuite.TagInfo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint@9.9.9", "--language", "python")
	cp.Expect("Could not find version 9.9.9 for package pylint")
	cp.ExpectExitCode(1)
}

func TestInfoIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InfoIntegrationTestSuite))
}
