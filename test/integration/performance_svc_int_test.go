package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

var SvcEnsureStartMaxTime = 1000 * time.Millisecond // https://activestatef.atlassian.net/browse/DX-935
var SvcRequestMaxTime = 50 * time.Millisecond
var SvcStopMaxTime = 50 * time.Millisecond

type PerformanceSvcIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PerformanceIntegrationTestSuite) TearDownSuite() {
	ipcClient := svcctl.NewDefaultIPCClient()
	err := svcctl.StopServer(ipcClient)
	suite.Require().NoError(err)
}

func (suite *PerformanceIntegrationTestSuite) TestSvcPerformance() {
	suite.OnlyRunForTags(tagsuite.Performance)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// This integration test is a bit special because it bypasses the spawning logic
	// so in order to get the right log files when debugging we manually provide the config dir
	var err error
	ts.Dirs.Config, err = storage.AppDataPath()
	suite.Require().NoError(err)

	ipcClient := svcctl.NewDefaultIPCClient()
	var svcPort string
	
	suite.Run("StartServer", func() {
		t := time.Now()
		svcPort, err = svcctl.EnsureExecStartedAndLocateHTTP(ipcClient, ts.SvcExe)
		suite.Require().NoError(err, ts.DebugMessage(fmt.Sprintf("Error: %s\nLog Tail:\n%s", errs.JoinMessage(err), logging.ReadTail())))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcEnsureStartMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service start took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})

	svcmodel := model.NewSvcModel(svcPort)
	svcmodel.EnableDebugLog()

	suite.Run("Query StateVersion", func() {
		t := time.Now()
		_, err := svcmodel.StateVersion(context.Background())
		suite.Require().NoError(err, ts.DebugMessage(fmt.Sprintf("Error: %s\nLog Tail:\n%s", errs.JoinMessage(err), logging.ReadTail())))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcRequestMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service request took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})

	suite.Run("Query Analytics", func() {
		t := time.Now()
		err := svcmodel.AnalyticsEvent(context.Background(), "performance-test", "performance-test", "performance-test", "{}")
		suite.Require().NoError(err, ts.DebugMessage(fmt.Sprintf("Error: %s\nLog Tail:\n%s", errs.JoinMessage(err), logging.ReadTail())))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcRequestMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service analytics request took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})

	suite.Run("Query Update", func() {
		t := time.Now()
		_, err := svcmodel.CheckUpdate(context.Background())
		suite.Require().NoError(err, ts.DebugMessage(fmt.Sprintf("Error: %s\nLog Tail:\n%s", errs.JoinMessage(err), logging.ReadTail())))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcRequestMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service update request took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})

	suite.Run("StopServer", func() {
		t := time.Now()
		err := svcctl.StopServer(ipcClient)
		suite.Require().NoError(err, ts.DebugMessage(fmt.Sprintf("Error: %s\nLog Tail:\n%s", errs.JoinMessage(err), logging.ReadTail())))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcStopMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service request took too long: %s (should be under %s)", duration.String(), SvcStopMaxTime.String()))
		}
	})
}

func TestPerformanceSvcIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceSvcIntegrationTestSuite))
}
