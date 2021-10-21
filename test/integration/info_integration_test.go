package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type InfoIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InfoIntegrationTestSuite) TestInfo_LatestVersion() {
	suite.OnlyRunForTags(tagsuite.Info)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint", "--language", "python")
	cp.Expect("Details for version 2.4.4")
	cp.Expect("Authors")
	cp.Expect("Python Code Quality Authority")
	cp.Expect("5 Version(s) Available")
	cp.Expect("2.4.4")
	cp.Expect("2.3.0")
	cp.Expect("0.28.0")
	cp.ExpectExitCode(0)
}

func (suite *InfoIntegrationTestSuite) TestInfo_SpecificVersion() {
	suite.OnlyRunForTags(tagsuite.Info)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("info", "pylint@0.28.0", "--language", "python")
	cp.Expect("Details for version 0.28.0")
	cp.Expect("Authors")
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
}

func TestInfoIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InfoIntegrationTestSuite))
}
