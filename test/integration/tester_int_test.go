package integration

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type TesterIntegrationTestSuite struct {
	integration.Suite
}

// TestActivatedEnv is a regression test for the following tickets:
// - https://www.pivotaltracker.com/story/show/167523128
// - https://www.pivotaltracker.com/story/show/169509213
func (suite *TesterIntegrationTestSuite) TestInActivatedEnv() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	defer ts.LogoutUser()

	p := suite.Spawn("activate")
	defer p.Close()

	p.Expect("Activating state: ActiveState-CLI/Python3")
	p.WaitForInput(10 * time.Second)

	p.SendLine(fmt.Sprintf("%s run test-interrupt", p.Executable()))
	p.Expect("Start of script", 5*time.Second)
	p.SendCtrlC()
	p.Expect("received interrupt", 3*time.Second)
	p.Expect("After first sleep or interrupt", 2*time.Second)
	p.SendCtrlC()
	suite.expectTerminateBatchJob(p)

	p.SendLine("exit 0")
	p.ExpectExitCode(0)
	suite.Require().NotContains(
		p.TrimmedSnapshot(), "not printed after second interrupt",
	)
}

func (suite *TesterIntegrationTestSuite) expectTerminateBatchJob(p *integration.Process) {
	if runtime.GOOS == "windows" {
		// send N to "Terminate batch job (Y/N)" question
		p.Expect("Terminate batch job")
		time.Sleep(200 * time.Millisecond)
		p.SendLine("N")
		p.Expect("N", 500*time.Millisecond)
	}
}

func TestTesterIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TesterIntegrationTestSuite))
}
