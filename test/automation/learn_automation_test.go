package automation

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
	"github.com/stretchr/testify/suite"
)

type LearnAutomationTestSuite struct {
	testsuite.Suite
}

func (suite *LearnAutomationTestSuite) TestLearn_UrlProvided() {
	suite.OnlyRunForTags(testsuite.TagAutomation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("learn")
	cp.ExpectLongString("https://platform.activestate.com/state-tool-cheat-sheet")
	cp.ExpectExitCode(0)
}

func TestLearnAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(LearnAutomationTestSuite))
}
