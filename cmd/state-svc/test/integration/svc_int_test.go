package integration

import (
	"syscall"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/suite"
)

type SvcIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *SvcIntegrationTestSuite) TestStartStop() {
	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("status"))
	cp.Expect("Service cannot be reached")
	cp.ExpectExitCode(1)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"))
	cp.Expect("Starting")
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("status"))
	cp.Expect("Checking")
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("stop"))
	cp.Expect("Stopping")
	cp.ExpectExitCode(0)
}

func (suite *SvcIntegrationTestSuite) TestSignals() {
	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// SIGINT (^C)
	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("foreground"))
	cp.Expect("Starting")
	cp.Signal(syscall.SIGINT)
	cp.Expect("caught a signal: interrupt")
	cp.ExpectNotExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("status"))
	cp.Expect("Service cannot be reached")
	cp.ExpectExitCode(1)

	// SIGTERM
	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("foreground"))
	cp.Expect("Starting")
	cp.Signal(syscall.SIGTERM)
	suite.NotContains(cp.TrimmedSnapshot(), "caught a signal")
	cp.ExpectNotExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("status"))
	cp.Expect("Service cannot be reached")
	cp.ExpectExitCode(1)
}

func (suite *SvcIntegrationTestSuite) TestSingleSvc() {
	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	oldCount := suite.GetNumStateSvcProcesses() // may be non-zero due to non-test state-svc processes
	for i := 1; i <= 10; i++ {
		go func() {
			ts.SpawnCmdWithOpts(ts.Exe, e2e.WithArgs("--version"))
		}()
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(1 * time.Second) // allow for some time to spawn the processes
	suite.Equal(oldCount+1, suite.GetNumStateSvcProcesses())
}

func (suite *SvcIntegrationTestSuite) GetNumStateSvcProcesses() int {
	procs, err := process.Processes()
	suite.NoError(err)

	count := 0
	for _, p := range procs {
		name, err := p.Name()
		suite.NoError(err)

		if svcName := constants.ServiceCommandName + exeutils.Extension; name == svcName {
			count++
		}
	}

	return count
}

func (suite *SvcIntegrationTestSuite) TestResolveRequestsBeforeStop() {
	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("foreground"))
	cp.Expect("Starting")

	cp2 := ts.SpawnCmdWithOpts(ts.Exe, e2e.WithArgs("update"))
	cp.Signal(syscall.SIGINT)
	cp2.Expect("Updating")
	cp2.Expect("Done")
	cp2.ExpectExitCode(0)
}

func TestSvcIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SvcIntegrationTestSuite))
}
