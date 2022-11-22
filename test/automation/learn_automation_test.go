package automation

import (
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
	"testing"
)

type LearnAutomationTestSuite struct {
	tagsuite.Suite
}

func (suite *LearnAutomationTestSuite) TestLearn_UrlProvided() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("learn")
	cp.ExpectLongString("https://platform.activestate.com/state-tool-cheat-sheet")
	cp.ExpectExitCode(0)
}

func TestLearnAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(LearnAutomationTestSuite))
}
