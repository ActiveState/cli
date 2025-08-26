package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ApiIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ApiIntegrationTestSuite) TestRequestHeaders() {
	suite.OnlyRunForTags(tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Empty", "."),
		e2e.OptAppendEnv(constants.DebugServiceRequestsEnvVarName+"=true", "VERBOSE=true"),
	)
	// e.g. User-Agent: state/0.38.0-SHA0deadbeef0; release (Windows; 10.0.22621; x86_64)
	cp.ExpectRe(`User-Agent: state/(\d+\.?)+-SHA[[:xdigit:]]+; \S+ \([^;]+; [^;]+; [^)]+\)`)
	cp.ExpectRe(`X-Requestor: [[:xdigit:]-]+`) // UUID
	cp.ExpectExitCode(0)
}

// TestNoApiCallsForPlainInvocation asserts that a bare `state` does not make any API calls.
func (suite *ApiIntegrationTestSuite) TestNoApiCallsForPlainInvocation() {
	suite.OnlyRunForTags(tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.OptAppendEnv(constants.DebugServiceRequestsEnvVarName + "=true"),
	)
	cp.ExpectExitCode(0)

	readLogFile := false
	for _, path := range ts.LogFiles() {
		if !strings.HasPrefix(filepath.Base(path), "state-") {
			continue
		}
		contents := string(fileutils.ReadFileUnsafe(path))
		suite.Assert().NotContains(contents, "URL: ") // pkg/platform/api logs URL, User-Agent, and X-Requestor for API calls
		readLogFile = true
	}
	suite.Assert().True(readLogFile, "did not read log file")
}

func (suite *ApiIntegrationTestSuite) TestAPIHostConfig_SetBeforeInvocation() {
	suite.OnlyRunForTags(tagsuite.API)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.SetConfig("api.host", "test.example.com")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "doesnt/exist"),
		e2e.OptAppendEnv("VERBOSE=true"),
	)
	cp.ExpectExitCode(11) // We know this command will fail, but we want to check the log file
	ts.IgnoreLogErrors()

	correctHostCount := 0
	incorrectHostCount := 0
	for _, path := range ts.LogFiles() {
		contents := string(fileutils.ReadFileUnsafe(path))
		if strings.Contains(contents, "test.example.com") {
			correctHostCount++
		}
		if strings.Contains(contents, "platform.activestate.com") {
			incorrectHostCount++
		}
	}
	suite.Assert().Greater(correctHostCount, 0, "Log file should contain the configured API host 'test.example.com'")
	suite.Assert().Equal(incorrectHostCount, 0, "Log file should not contain the default API host 'platform.activestate.com'")

	// Clean up - remove the config setting
	cp = ts.Spawn("config", "set", "api.host", "")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)
}

func (suite *ApiIntegrationTestSuite) TestAPIHostConfig_SetOnFirstInvocation() {
	suite.OnlyRunForTags(tagsuite.API)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", "api.host", "test.example.com")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "doesnt/exist"),
		e2e.OptAppendEnv("VERBOSE=true"),
	)
	cp.ExpectExitCode(11) // We know this command will fail, but we want to check the log file
	ts.IgnoreLogErrors()

	// Because the config value is set on first invocation of the state tool the state-svc will start
	// before the state tool has a chance to set the host in the config. This means that it will still
	// use the default host. The above test confirms that the service will use the configured host if
	// the config is set before the state tool is invoked.
	correctHostCount := 0
	for _, path := range ts.LogFiles() {
		contents := string(fileutils.ReadFileUnsafe(path))
		if strings.Contains(contents, "test.example.com") {
			correctHostCount++
		}
	}
	suite.Assert().Greater(correctHostCount, 0, "Log file should contain the configured API host 'test.example.com'")

	// Clean up - remove the config setting
	cp = ts.Spawn("config", "set", "api.host", "")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)
}

func TestApiIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ApiIntegrationTestSuite))
}
