package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"
)

type HelpIntegrationTestSuite struct {
	tagsuite.Suite
}

func TestHelpIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HelpIntegrationTestSuite))
}

func (suite *HelpIntegrationTestSuite) TestCommandListing() {
	suite.OnlyRunForTags(tagsuite.Help)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.OptArgs("--help"), e2e.OptTermTest(termtest.OptCols(80)))
	cp.Expect("Usage:")
	cp.Expect("Environment Setup:")
	cp.Expect("Environment Usage:")
	cp.Expect("Project Usage:")
	cp.Expect("Package Management:")
	cp.Expect("Platform:")
	cp.Expect("Version Control:")
	cp.Expect("Automation:")
	cp.Expect("Utilities:")
	cp.Expect("    remove") // wrapped on word, not character
	cp.Expect("Flags:")
}
