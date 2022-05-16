package integration

import (
	"net"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/svcctl"
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
	time.Sleep(500 * time.Millisecond) // wait for service to start up

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("status"))
	cp.Expect("Checking")

	// Verify it created a socket file.
	sockFile := svcctl.NewIPCSockPathFromGlobals().String()
	suite.True(fileutils.TargetExists(sockFile))

	// Verify the server is running on its reported port.
	cp.ExpectRe("Port:\\s+:\\d+")
	portRe := regexp.MustCompile("Port:\\s+:(\\d+)")
	port := portRe.FindStringSubmatch(cp.TrimmedSnapshot())[1]
	_, err := net.Listen("tcp", "localhost:"+port)
	suite.Error(err)

	// Verify it created and wrote to its reported log file.
	cp.ExpectRe("Log:\\s+.+?\\.log")
	logRe := regexp.MustCompile("Log:\\s+(.+?\\.log)")
	logFile := logRe.FindStringSubmatch(cp.TrimmedSnapshot())[1]
	suite.True(fileutils.FileExists(logFile))
	suite.True(len(fileutils.ReadFileUnsafe(logFile)) > 0)

	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("stop"))
	cp.Expect("Stopping")
	cp.ExpectExitCode(0)
	time.Sleep(500 * time.Millisecond) // wait for service to stop

	// Verify it deleted its socket file.
	suite.False(fileutils.TargetExists(sockFile))
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

	sockFile := svcctl.NewIPCSockPathFromGlobals().String()
	suite.False(fileutils.TargetExists(sockFile))

	// SIGTERM
	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("foreground"))
	cp.Expect("Starting")
	cp.Signal(syscall.SIGTERM)
	suite.NotContains(cp.TrimmedSnapshot(), "caught a signal")
	cp.ExpectNotExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("status"))
	cp.Expect("Service cannot be reached")
	cp.ExpectExitCode(1)

	suite.False(fileutils.TargetExists(sockFile))
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
	time.Sleep(2 * time.Second) // allow for some time to spawn the processes
	suite.Equal(oldCount+1, suite.GetNumStateSvcProcesses())
}

func (suite *SvcIntegrationTestSuite) GetNumStateSvcProcesses() int {
	procs, err := process.Processes()
	suite.NoError(err)

	count := 0
	for _, p := range procs {
		name, err := p.Name()
		suite.NoError(err)
		name = filepath.Base(name) // just in case an absolute path is returned

		if svcName := constants.ServiceCommandName + exeutils.Extension; name == svcName {
			count++
		}
	}

	return count
}

func TestSvcIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SvcIntegrationTestSuite))
}
