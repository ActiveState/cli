package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/analytics/client/sync/reporters"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
)

type AnalyticsIntegrationTestSuite struct {
	tagsuite.Suite
	eventsfile string
}

// TestActivateEvents ensures that the right events are sent when we activate
// Note the heartbeat code especially is a little awkward as we have to account for timing offsets between state and
// state-svc. For that reason we tend to assert "greater than" rather than equals, because checking for equals introduces
// race conditions into the testing suite (not the state tool itself).
func (suite *AnalyticsIntegrationTestSuite) TestActivateEvents() {
	suite.OnlyRunForTags(tagsuite.Analytics, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	// We want to do a clean test without an activate event, so we have to manually seed the yaml
	url := "https://platform.activestate.com/ActiveState-CLI/Alternate-Python?branch=main&commitID=efcc851f-1451-4d0a-9dcb-074ac3f35f0a"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	heartbeatInterval := 1000 // in milliseconds
	sleepTime := time.Duration(heartbeatInterval) * time.Millisecond
	sleepTime = sleepTime + (sleepTime / 2)

	env := []string{
		constants.DisableRuntime + "=false",
		fmt.Sprintf("%s=%d", constants.HeartbeatIntervalEnvVarName, heartbeatInterval),
	}

	var cp *termtest.ConsoleProcess
	if runtime.GOOS == "windows" {
		cp = ts.SpawnCmdWithOpts("cmd.exe",
			e2e.WithArgs("/k", "state", "activate"),
			e2e.WithWorkDirectory(ts.Dirs.Work),
			e2e.AppendEnv(env...),
		)
	} else {
		cp = ts.SpawnWithOpts(e2e.WithArgs("activate"),
			e2e.WithWorkDirectory(ts.Dirs.Work),
			e2e.AppendEnv(env...),
		)
	}

	cp.Expect("Creating a Virtual Environment")
	cp.Expect("Activated")
	cp.WaitForInput(120 * time.Second)

	time.Sleep(time.Second) // Ensure state-svc has time to report events

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	events := suite.parseEvents(ts)
	suite.Require().NotEmpty(events)

	// Runtime:start events
	suite.assertNEvents(events, 1, anaConst.CatRuntime, anaConst.ActRuntimeStart,
		fmt.Sprintf("output:\n%s\nState Log:\n%s\nSvc Log:\n%s",
			cp.Snapshot(), ts.MostRecentStateLog(), ts.SvcLog()))

	// Runtime:success events
	suite.assertNEvents(events, 1, anaConst.CatRuntime, anaConst.ActRuntimeSuccess,
		fmt.Sprintf("output:\n%s\nState Log:\n%s\nSvc Log:\n%s",
			cp.Snapshot(), ts.MostRecentStateLog(), ts.SvcLog()))

	heartbeatInitialCount := suite.countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)
	if heartbeatInitialCount < 2 {
		// It's possible due to the timing of the heartbeats and the fact that they are async that we have gotten either
		// one or two by this point. Technically more is possible, just very unlikely.
		suite.Fail(fmt.Sprintf("Received %d heartbeats, realistically we should at least have gotten 2", heartbeatInitialCount))
	}

	time.Sleep(sleepTime)

	events = suite.parseEvents(ts)
	suite.Require().NotEmpty(events)

	// Runtime-use:heartbeat events - should now be at least +1 because we waited <heartbeatInterval>
	suite.assertGtEvents(events, heartbeatInitialCount, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat,
		fmt.Sprintf("output:\n%s\nState Log:\n%s\nSvc Log:\n%s",
			cp.Snapshot(), ts.MostRecentStateLog(), ts.SvcLog()))

	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	time.Sleep(sleepTime) // give time to let rtwatcher detect process has exited

	events = suite.parseEvents(ts)
	suite.Require().NotEmpty(events)
	eventsAfterExit := suite.countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)

	time.Sleep(sleepTime)

	events = suite.parseEvents(ts)
	suite.Require().NotEmpty(events)
	eventsAfterWait := suite.countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)

	suite.Equal(eventsAfterExit, eventsAfterWait,
		fmt.Sprintf("Heartbeats should stop ticking after exiting subshell.\n"+
			"output:\n%s\nState Log:\n%s\nSvc Log:\n%s",
			cp.Snapshot(), ts.MostRecentStateLog(), ts.SvcLog()))

	// Ensure any analytics events from the state tool have the instance ID set
	for _, e := range events {
		if strings.Contains(e.Category, "state-svc") || strings.Contains(e.Action, "state-svc") {
			continue
		}
		suite.NotEmpty(e.Dimensions.InstanceID)
	}

	suite.assertSequentialEvents(events)
}

func (suite *AnalyticsIntegrationTestSuite) countEvents(events []reporters.TestLogEntry, category, action string) int {
	filteredEvents := funk.Filter(events, func(e reporters.TestLogEntry) bool {
		return e.Category == category && e.Action == action
	}).([]reporters.TestLogEntry)
	return len(filteredEvents)
}

func (suite *AnalyticsIntegrationTestSuite) assertNEvents(events []reporters.TestLogEntry,
	expectedN int, category, action string, errMsg string) {
	suite.Assert().Equal(expectedN, suite.countEvents(events, category, action),
		"Expected %d %s:%s events.\nFile location: %s\nEvents received:\n%s\nError:\n%s",
		expectedN, category, action, suite.eventsfile, suite.summarizeEvents(events), errMsg)
}

func (suite *AnalyticsIntegrationTestSuite) assertGtEvents(events []reporters.TestLogEntry,
	greaterThanN int, category, action string, errMsg string) {
	suite.Assert().Greater(suite.countEvents(events, category, action), greaterThanN,
		fmt.Sprintf("Expected more than %d %s:%s events.\nFile location: %s\nEvents received:\n%s\nError:\n%s",
			greaterThanN, category, action, suite.eventsfile, suite.summarizeEvents(events), errMsg))
}

func (suite *AnalyticsIntegrationTestSuite) assertSequentialEvents(events []reporters.TestLogEntry) {
	seq := map[string]int{}

	// Since sequence is established client-side and then reported async it's possible that the sequence does not match the
	// slice ordering of events
	sort.Slice(events, func(i, j int) bool {
		return *events[i].Dimensions.Sequence < *events[j].Dimensions.Sequence
	})

	var lastEvent reporters.TestLogEntry
	for _, ev := range events {
		if *ev.Dimensions.Sequence == -1 {
			continue // The sequence of this event is irrelevant
		}
		// Sequence is per instance ID
		key := (*ev.Dimensions.InstanceID)[0:6]
		if v, ok := seq[key]; ok {
			if (v + 1) != *ev.Dimensions.Sequence {
				suite.Failf(fmt.Sprintf("Events are not sequential, expected %d but got %d", v+1, *ev.Dimensions.Sequence),
					suite.summarizeEventSequence([]reporters.TestLogEntry{
						lastEvent, ev,
					}))
			}
		} else {
			if *ev.Dimensions.Sequence != 0 {
				suite.Fail(fmt.Sprintf("Sequence should start at 0, got: %v\nevents:\n %v",
					suite.summarizeEventSequence([]reporters.TestLogEntry{ev}),
					suite.summarizeEventSequence(events)))
			}
		}
		seq[key] = *ev.Dimensions.Sequence
		lastEvent = ev
	}
}

func (suite *AnalyticsIntegrationTestSuite) summarizeEvents(events []reporters.TestLogEntry) string {
	summary := []string{}
	for _, event := range events {
		summary = append(summary, fmt.Sprintf("%s:%s:%s", event.Category, event.Action, event.Label))
	}
	return strings.Join(summary, "\n")
}

func (suite *AnalyticsIntegrationTestSuite) summarizeEventSequence(events []reporters.TestLogEntry) string {
	summary := []string{}
	for _, event := range events {
		summary = append(summary, fmt.Sprintf("%s:%s:%s (seq: %s:%s:%d)\n",
			event.Category, event.Action, event.Label,
			*event.Dimensions.Command, (*event.Dimensions.InstanceID)[0:6], *event.Dimensions.Sequence))
	}
	return strings.Join(summary, "\n")
}

func (suite *AnalyticsIntegrationTestSuite) parseEvents(s *e2e.Session) []reporters.TestLogEntry {
	time.Sleep(time.Second) // give svc time to process events

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
	events := suite.parseEvents(ts)

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

func (suite *AnalyticsIntegrationTestSuite) TestSend() {
	suite.OnlyRunForTags(tagsuite.Analytics, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	cp := ts.Spawn("--version")
	cp.Expect("Version")
	cp.ExpectExitCode(0)

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	cp = ts.Spawn("config", "set", constants.ReportAnalyticsConfig, "false")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)

	initialEvents := suite.parseEvents(ts)
	suite.assertSequentialEvents(initialEvents)

	cp = ts.Spawn("--version")
	cp.Expect("Version")
	cp.ExpectExitCode(0)

	events := suite.parseEvents(ts)
	currentEvents := len(events)
	if currentEvents > len(initialEvents) {
		suite.Failf("Should not get additional events", "Got %d additional events, should be 0", currentEvents-len(initialEvents))
	}

	suite.assertSequentialEvents(events)
}

func (suite *AnalyticsIntegrationTestSuite) TestSequenceAndFlags() {
	suite.OnlyRunForTags(tagsuite.Analytics)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	cp := ts.Spawn("--version")
	cp.Expect("Version")
	cp.ExpectExitCode(0)

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
	events := suite.parseEvents(ts)
	suite.assertSequentialEvents(events)

	found := false
	for _, ev := range events {
		if ev.Category == "run-command" && ev.Action == "" && ev.Label == "--version" {
			found = true
			break
		}
	}

	suite.True(found, "Should have run-command event with flags, actual: %s", suite.summarizeEvents(events))
}

func TestAnalyticsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AnalyticsIntegrationTestSuite))
}
