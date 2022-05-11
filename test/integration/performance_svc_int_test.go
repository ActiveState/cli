package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

var SvcEnsureStartMaxTime = 1000 * time.Millisecond // https://activestatef.atlassian.net/browse/DX-935
var SvcRequestMaxTime = 50 * time.Millisecond
var SvcStopMaxTime = 10 * time.Millisecond

type PerformanceSvcIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PerformanceIntegrationTestSuite) AfterTest(suiteName, testName string) {
	ipcClient := svcctl.NewDefaultIPCClient()
	err := svcctl.StopServer(ipcClient)
	suite.Require().NoError(err)
}

func (suite *PerformanceIntegrationTestSuite) TestSvcPerformance() {
	suite.OnlyRunForTags(tagsuite.Performance)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ipcClient := svcctl.NewDefaultIPCClient()
	var svcPort string

	suite.Run("StartServer", func() {
		svcExec, err := installation.ServiceExecFromDir(ts.Dirs.Bin)
		suite.Require().NoError(err, errs.JoinMessage(err))

		t := time.Now()
		svcPort, err = svcctl.EnsureExecStartedAndLocateHTTP(ipcClient, svcExec)
		suite.Require().NoError(err, errs.JoinMessage(err))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcEnsureStartMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service start took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})

	suite.Run("Query StateVersion", func() {
		t := time.Now()
		svcmodel := model.NewSvcModel(svcPort)
		_, err := svcmodel.StateVersion(context.Background())
		suite.Require().NoError(err, errs.JoinMessage(err))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcRequestMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service request took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})

	suite.Run("Query Analytics", func() {
		t := time.Now()
		svcmodel := model.NewSvcModel(svcPort)
		err := svcmodel.AnalyticsEvent(context.Background(), "performance-test", "performance-test", "performance-test", "{}")
		suite.Require().NoError(err, errs.JoinMessage(err), ts.DebugMessage(errs.JoinMessage(err)))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcRequestMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service analytics request took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})

	suite.Run("Query Update", func() {
		t := time.Now()
		svcmodel := model.NewSvcModel(svcPort)
		_, err := svcmodel.CheckUpdate(context.Background())
		suite.Require().NoError(err, ts.DebugMessage(errs.JoinMessage(err)))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcRequestMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service update request took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})

	suite.Run("StopServer", func() {
		t := time.Now()
		err := svcctl.StopServer(ipcClient)
		suite.Require().NoError(err, errs.JoinMessage(err), ts.DebugMessage(errs.JoinMessage(err)))
		duration := time.Since(t)

		if duration.Nanoseconds() > SvcStopMaxTime.Nanoseconds() {
			suite.Fail(fmt.Sprintf("Service request took too long: %s (should be under %s)", duration.String(), SvcEnsureStartMaxTime.String()))
		}
	})
}

func TestPerformanceSvcIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceSvcIntegrationTestSuite))
}
