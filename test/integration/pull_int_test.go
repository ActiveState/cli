package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

type PullIntegrationTestSuite struct {
	suite.Suite
}

func (suite *PullIntegrationTestSuite) TestPull() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/Python3"`)

	cp := ts.Spawn("pull")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("pull")
	cp.Expect("already up to date")
	cp.ExpectExitCode(0)
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
