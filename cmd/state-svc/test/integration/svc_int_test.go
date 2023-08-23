package integration

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/suite"
)

type SvcIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *SvcIntegrationTestSuite) TestStartStop() {
	// Disable test until we can fix console output on Windows
	// See issue here: https://activestatef.atlassian.net/browse/DX-1311
	suite.T().SkipNow()
	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("stop"))
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("status"))
	cp.Expect("Service cannot be reached")
	cp.ExpectExitCode(1)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("start"))
	cp.Expect("Starting")
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("status"))
	cp.Expect("Checking")

	// Verify the server is running on its reported port.
	cp.ExpectRe("Port:\\s+:\\d+\\s")
	portRe := regexp.MustCompile("Port:\\s+:(\\d+)")
	port := portRe.FindStringSubmatch(cp.Output())[1]
	_, err := net.Listen("tcp", "localhost:"+port)
	suite.Error(err)

	// Verify it created and wrote to its reported log file.
	cp.ExpectRe("Log:\\s+.+?\\.log")
	logRe := regexp.MustCompile("Log:\\s+(.+?\\.log)")
	logFile := logRe.FindStringSubmatch(cp.Output())[1]
	suite.True(fileutils.FileExists(logFile), "log file '"+logFile+"' does not exist")
	suite.True(len(fileutils.ReadFileUnsafe(logFile)) > 0, "log file is empty")

	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("stop"))
	cp.Expect("Stopping")
	cp.ExpectExitCode(0)
	time.Sleep(500 * time.Millisecond) // wait for service to stop

	// Verify the port is free.
	server, err := net.Listen("tcp", "localhost:"+port)
	suite.NoError(err)
	server.Close()
}

func (suite *SvcIntegrationTestSuite) TestSignals() {
	if condition.OnCI() {
		// https://activestatef.atlassian.net/browse/DX-964
		// https://activestatef.atlassian.net/browse/DX-980
		suite.T().Skip("Signal handling on CI is unstable and unreliable")
	}

	if runtime.GOOS == "windows" {
		suite.T().Skip("Windows does not support signal sending.")
	}

	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// SIGINT (^C)
	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("foreground"))
	cp.Expect("Starting")
	time.Sleep(1 * time.Second) // wait for the service to start up
	cp.Cmd().Process.Signal(syscall.SIGINT)
	cp.Expect("caught a signal: interrupt")
	cp.ExpectNotExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("status"))
	cp.Expect("Service cannot be reached")
	cp.ExpectExitCode(1)

	sockFile := svcctl.NewIPCSockPathFromGlobals().String()
	suite.False(fileutils.TargetExists(sockFile), "socket file was not deleted")

	// SIGTERM
	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("foreground"))
	cp.Expect("Starting")
	time.Sleep(1 * time.Second) // wait for the service to start up
	cp.Cmd().Process.Signal(syscall.SIGTERM)
	suite.NotContains(cp.Output(), "caught a signal")
	cp.ExpectExitCode(0) // should exit gracefully

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("status"))
	cp.Expect("Service cannot be reached")
	cp.ExpectExitCode(1)

	suite.False(fileutils.TargetExists(sockFile), "socket file was not deleted")
}

func (suite *SvcIntegrationTestSuite) TestStartDuplicateErrorOutput() {
	// https://activestatef.atlassian.net/browse/DX-1136
	suite.OnlyRunForTags(tagsuite.Service)
	if runtime.GOOS == "windows" {
		suite.T().Skip("Windows doesn't seem to read from svc at the moment")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("stop"))
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("status"))
	cp.ExpectNotExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("start"))
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("foreground"))
	cp.Expect("An existing server instance appears to be in use")
	cp.ExpectExitCode(1)

	cp = ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("stop"))
	cp.ExpectExitCode(0)
}

func (suite *SvcIntegrationTestSuite) TestSingleSvc() {
	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.SpawnCmdWithOpts(ts.SvcExe, e2e.OptArgs("stop"))
	time.Sleep(2 * time.Second) // allow for some time to stop the existing available process

	oldCount := suite.GetNumStateSvcProcesses() // may be non-zero due to non-test state-svc processes (using different sock file)
	for i := 1; i <= 10; i++ {
		go ts.SpawnCmdWithOpts(ts.Exe, e2e.OptArgs("--version"))
		time.Sleep(50 * time.Millisecond) // do not spam CPU
	}
	time.Sleep(2 * time.Second) // allow for some time to spawn the processes

	for attempts := 100; attempts > 0; attempts-- {
		suite.T().Log("iters left:", attempts, "procs:", suite.GetNumStateSvcProcesses())
		if suite.GetNumStateSvcProcesses() == oldCount+1 {
			break
		}
		time.Sleep(2 * time.Second) // keep waiting
	}

	newCount := suite.GetNumStateSvcProcesses()
	if newCount > oldCount+1 {
		// We only care if we end up with more services than anticipated. We can actually end up with less than we started
		// with due to other integration tests not always waiting for state-svc to have fully shut down before running the next test
		suite.Fail(fmt.Sprintf("spawning multiple state processes should only result in one more state-svc process at most, newCount: %d, oldCount: %d", newCount, oldCount))
	}
}

func (suite *SvcIntegrationTestSuite) GetNumStateSvcProcesses() int {
	procs, err := process.Processes()
	suite.NoError(err)

	count := 0
	for _, p := range procs {
		if name, err := p.Name(); err == nil {
			name = filepath.Base(name) // just in case an absolute path is returned
			if svcName := constants.ServiceCommandName + exeutils.Extension; name == svcName {
				count++
			}
		}
	}

	return count
}

func (suite *SvcIntegrationTestSuite) TestAutostartConfigEnableDisable() {
	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Toggle it via state tool config.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("config", "set", constants.AutostartSvcConfigKey, "false"),
	)
	cp.ExpectExitCode(0)
	cp = ts.SpawnWithOpts(e2e.OptArgs("config", "get", constants.AutostartSvcConfigKey))
	cp.Expect("false")
	cp.ExpectExitCode(0)

	// Toggle it again via state tool config.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("config", "set", constants.AutostartSvcConfigKey, "true"),
	)
	cp.ExpectExitCode(0)
	cp = ts.SpawnWithOpts(e2e.OptArgs("config", "get", constants.AutostartSvcConfigKey))
	cp.Expect("true")
	cp.ExpectExitCode(0)
}

func (suite *SvcIntegrationTestSuite) TestLogRotation() {
	suite.OnlyRunForTags(tagsuite.Service)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	logDir := filepath.Join(ts.Dirs.Config, "logs")

	// Create a tranche of 30-day old dummy log files.
	numFooFiles := 50
	thirtyDaysOld := time.Now().Add(-24 * time.Hour * 30)
	for i := 1; i <= numFooFiles; i++ {
		logFile := filepath.Join(logDir, fmt.Sprintf("foo-%d%s", i, logging.FileNameSuffix))
		err := fileutils.Touch(logFile)
		suite.Require().NoError(err, "could not create dummy log file")
		err = os.Chtimes(logFile, thirtyDaysOld, thirtyDaysOld)
		suite.Require().NoError(err, "must be able to change file modification times")
	}

	// Override state-svc log rotation interval from 1 minute to 4 seconds for this test.
	logRotateInterval := 4 * time.Second
	os.Setenv(constants.SvcLogRotateIntervalEnvVarName, fmt.Sprintf("%d", logRotateInterval.Milliseconds()))
	defer os.Unsetenv(constants.SvcLogRotateIntervalEnvVarName)

	// We want the side-effect of spawning state-svc.
	cp := ts.Spawn("--version")
	cp.Expect("ActiveState CLI")
	cp.ExpectExitCode(0)

	initialWait := 2 * time.Second
	time.Sleep(initialWait) // wait for state-svc to perform initial log rotation

	// Verify the log rotation pruned the dummy log files.
	files, err := ioutil.ReadDir(logDir)
	suite.Require().NoError(err)
	remainingFooFiles := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "foo-") {
			remainingFooFiles++
		}
	}
	suite.Less(remainingFooFiles, numFooFiles, "no foo.log files were cleaned up; expected at least one to be")

	// state-svc is still running, along with its log rotation timer.
	// Re-create another tranche of 30-day old dummy log files for when the timer fires again.
	numFooFiles += remainingFooFiles
	for i := remainingFooFiles + 1; i <= numFooFiles; i++ {
		logFile := filepath.Join(logDir, fmt.Sprintf("foo-%d%s", i, logging.FileNameSuffix))
		err := fileutils.Touch(logFile)
		suite.Require().NoError(err, "could not create dummy log file")
		err = os.Chtimes(logFile, thirtyDaysOld, thirtyDaysOld)
		suite.Require().NoError(err, "must be able to change file modification times")
	}

	time.Sleep(logRotateInterval - initialWait) // wait for another log rotation

	// Verify that another log rotation pruned the dummy log files.
	files, err = ioutil.ReadDir(logDir)
	suite.Require().NoError(err)
	remainingFooFiles = 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "foo-") {
			remainingFooFiles++
		}
	}
	suite.Less(remainingFooFiles, numFooFiles, "no more foo.log files were cleaned up (on timer); expected at least one to be")
}

func TestSvcIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SvcIntegrationTestSuite))
}
