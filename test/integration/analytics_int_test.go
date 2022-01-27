package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/analytics/client/sync/reporters"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
)

type AnalyticsIntegrationTestSuite struct {
	tagsuite.Suite
	eventsfile string
}

func (suite *AnalyticsIntegrationTestSuite) TestActivateEvents() {
	suite.OnlyRunForTags(tagsuite.Analytics, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	// // We want to do a clean test without an activate event, so we have to manually seed the yaml
	url := "https://platform.activestate.com/ActiveState-CLI/Alternate-Python?branch=main&commitID=efcc851f-1451-4d0a-9dcb-074ac3f35f0a"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	heartbeatInterval := 5000 // in milliseconds

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.WithWorkDirectory(ts.Dirs.Work),
		e2e.AppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
			fmt.Sprintf("%s=%d", constants.HeartbeatIntervalEnvVarName, heartbeatInterval),
		),
	)

	cp.Expect("Creating a Virtual Environment")
	cp.Expect("Activated")
	cp.WaitForInput(120 * time.Second)

	time.Sleep(time.Second) // Ensure state-svc has time to report events

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	events := suite.parseEvents()
	suite.Require().NotEmpty(events)

	// Runtime:start events
	suite.assertNEvents(events, 1, anaConst.CatRuntime, anaConst.ActRuntimeStart)

	// Runtime:success events
	suite.assertNEvents(events, 1, anaConst.CatRuntime, anaConst.ActRuntimeSuccess)

	heartbeatInitialCount := suite.countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)
	if heartbeatInitialCount > 2 {
		// It's possible due to the timing of the heartbeats and the fact that they are async that we have gotten either
		// one or two by this point. Technically more is possible, just very unlikely.
		suite.Fail("Received %d heartbeats, realistically we should at most have gotten 2", heartbeatInitialCount)
	}

	// Runtime-use:heartbeat events
	suite.assertNEvents(events, heartbeatInitialCount, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)

	time.Sleep(time.Duration(heartbeatInterval) * time.Millisecond)

	events = suite.parseEvents()
	suite.Require().NotEmpty(events)

	// Runtime-use:heartbeat events - should now be +1 because we waited <heartbeatInterval>
	suite.assertNEvents(events, heartbeatInitialCount+1, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)

	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	time.Sleep((time.Duration(heartbeatInterval) * time.Millisecond))

	events = suite.parseEvents()
	suite.Require().NotEmpty(events)

	// Runtime-use:heartbeat events - should still be +1 because we exited the process so it's no longer using the runtime
	suite.assertNEvents(events, heartbeatInitialCount+1, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)
}

func (suite *AnalyticsIntegrationTestSuite) countEvents(events []reporters.TestLogEntry, category, action string) int {
	filteredEvents := funk.Filter(events, func(e reporters.TestLogEntry) bool {
		return e.Category == category && e.Action == action
	}).([]reporters.TestLogEntry)
	return len(filteredEvents)
}

func (suite *AnalyticsIntegrationTestSuite) assertNEvents(events []reporters.TestLogEntry, expectedN int, category, action string) {
	suite.Assert().Equal(expectedN, suite.countEvents(events, category, action),
		"Expected %d %s:%s events.\nFile location: %s\nEvents received:\n%s", expectedN, category, action, suite.eventsfile, suite.summarizeEvents(events))
}

func (suite *AnalyticsIntegrationTestSuite) summarizeEvents(events []reporters.TestLogEntry) string {
	summary := []string{}
	for _, event := range events {
		summary = append(summary, fmt.Sprintf("%s:%s:%s", event.Category, event.Action, event.Label))
	}
	return strings.Join(summary, "\n")
}

func (suite *AnalyticsIntegrationTestSuite) parseEvents() []reporters.TestLogEntry {
	suite.Require().FileExists(suite.eventsfile)

	b, err := fileutils.ReadFile(suite.eventsfile)
	suite.Require().NoError(err)

	var result []reporters.TestLogEntry
	entries := strings.Split(string(b), "\x00")
	for _, entry := range entries {
		if len(entry) == 0 {
			continue
		}

		var parsedEntry reporters.TestLogEntry
		err := json.Unmarshal([]byte(entry), &parsedEntry)
		suite.Require().NoError(err, fmt.Sprintf("path: %s, value: \n%s\n", suite.eventsfile, entry))
		result = append(result, parsedEntry)
	}

	return result
}

func (suite *AnalyticsIntegrationTestSuite) TestShim() {
	suite.OnlyRunForTags(tagsuite.Analytics)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	asyData := strings.TrimSpace(`
project: https://platform.activestate.com/ActiveState-CLI/test?commitID=9090c128-e948-4388-8f7f-96e2c1e00d98
scripts:
  - name: pip
    language: bash
    standalone: true
    value: echo "pip"
`)

	ts.PrepareActiveStateYAML(asyData)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/Alternate-Python"),
		e2e.WithWorkDirectory(ts.Dirs.Work),
		e2e.AppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
		),
	)

	cp.Expect("Creating a Virtual Environment")
	cp.Expect("Activated")
	cp.WaitForInput(120 * time.Second)

	cp = ts.Spawn("run", "pip")
	cp.Wait()

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
	events := suite.parseEvents()

	var found int
	for _, event := range events {
		if event.Category == anaConst.CatRunCmd && event.Action == "run" {
			found++
			suite.Equal(constants.PipShim, event.Label)
		}
	}

	if found <= 0 {
		suite.Fail("Did not find shim event")
	}
}

func TestAnalyticsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AnalyticsIntegrationTestSuite))
}
