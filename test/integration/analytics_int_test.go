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

	"github.com/ActiveState/termtest"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/analytics/client/sync/reporters"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	helperSuite "github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type AnalyticsIntegrationTestSuite struct {
	tagsuite.Suite
	eventsfile string
}

// TestHeartbeats ensures that heartbeats are send on runtime use
// Note the heartbeat code especially is a little awkward as we have to account for timing offsets between state and
// state-svc. For that reason we tend to assert "greater than" rather than equals, because checking for equals introduces
// race conditions into the testing suite (not the state tool itself).
func (suite *AnalyticsIntegrationTestSuite) TestHeartbeats() {
	suite.OnlyRunForTags(tagsuite.Analytics, tagsuite.Critical)

	/* TEST SETUP */

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "ActiveState-CLI/Alternate-Python"
	commitID := "efcc851f-1451-4d0a-9dcb-074ac3f35f0a"

	// We want to do a clean test without an activate event, so we have to manually seed the yaml
	ts.PrepareProject(namespace, commitID)

	heartbeatInterval := 1000 // in milliseconds
	sleepTime := time.Duration(heartbeatInterval) * time.Millisecond
	sleepTime = sleepTime + (sleepTime / 2)

	env := []string{
		fmt.Sprintf("%s=%d", constants.HeartbeatIntervalEnvVarName, heartbeatInterval),
	}

	/* ACTIVATE TESTS */

	// Produce Activate Heartbeats

	var cp *e2e.SpawnedCmd
	if runtime.GOOS == "windows" {
		cp = ts.SpawnCmdWithOpts("cmd.exe",
			e2e.OptArgs("/k", "state", "activate"),
			e2e.OptWD(ts.Dirs.Work),
			e2e.OptAppendEnv(env...),
		)
	} else {
		cp = ts.SpawnWithOpts(e2e.OptArgs("activate"),
			e2e.OptWD(ts.Dirs.Work),
			e2e.OptAppendEnv(env...),
		)
	}

	cp.Expect("Creating a Virtual Environment")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput()

	time.Sleep(time.Second) // Ensure state-svc has time to report events

	// By this point the activate heartbeats should have been recorded

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	events := parseAnalyticsEvents(suite, ts)

	// Now it's time for us to assert that we are seeing the expected number of events

	suite.Require().NotEmpty(events)

	// Runtime:start events
	suite.assertNEvents(events, 1, anaConst.CatRuntimeDebug, anaConst.ActRuntimeStart, anaConst.SrcStateTool,
		fmt.Sprintf("output:\n%s\n%s",
			cp.Output(), ts.DebugLogsDump()))

	// Runtime:success events
	suite.assertNEvents(events, 1, anaConst.CatRuntimeDebug, anaConst.ActRuntimeSuccess, anaConst.SrcStateTool,
		fmt.Sprintf("output:\n%s\n%s",
			cp.Output(), ts.DebugLogsDump()))

	// Runtime-use:attempts events
	attemptInitialCount := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeAttempt, anaConst.SrcStateTool)
	suite.Equal(1, attemptInitialCount, "Activate should have resulted in 1 attempt")

	// Runtime-use:heartbeat events
	heartbeatInitialCount := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, anaConst.SrcStateTool)
	if heartbeatInitialCount < 2 {
		// It's possible due to the timing of the heartbeats and the fact that they are async that we have gotten either
		// one or two by this point. Technically more is possible, just very unlikely.
		suite.Fail(fmt.Sprintf("Received %d heartbeats, realistically we should at least have gotten 2", heartbeatInitialCount))
	}

	// Wait for additional heartbeats to be reported, because our activated shell is still open

	time.Sleep(sleepTime)

	events = parseAnalyticsEvents(suite, ts)
	suite.Require().NotEmpty(events)

	suite.assertNEvents(events, 1, anaConst.CatRuntimeUsage, anaConst.ActRuntimeAttempt, anaConst.SrcStateTool, "Should still only have 1 attempt")

	// Runtime-use:heartbeat events - should now be at least +1 because we waited <heartbeatInterval>
	suite.assertGtEvents(events, heartbeatInitialCount, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, anaConst.SrcStateTool,
		fmt.Sprintf("output:\n%s\n%s",
			cp.Output(), ts.DebugLogsDump()))

	/* EXECUTOR TESTS */

	// Test that executor is sending heartbeats
	suite.Run("Executors", func() {
		cp.SendLine("python3 -c \"import sys; print(sys.copyright)\"")
		cp.Expect("provided by ActiveState")

		time.Sleep(sleepTime)

		eventsAfterExecutor := parseAnalyticsEvents(suite, ts)
		suite.Require().Greater(len(eventsAfterExecutor), len(events), "Should have received more events after running executor")

		executorEvents := filterEvents(eventsAfterExecutor, func(e reporters.TestLogEntry) bool {
			if e.Dimensions == nil || e.Dimensions.Trigger == nil {
				return false
			}
			return (*e.Dimensions.Trigger) == trigger.TriggerExecutor.String()
		})
		suite.Require().Equal(1, countEvents(executorEvents, anaConst.CatRuntimeUsage, anaConst.ActRuntimeAttempt, anaConst.SrcExecutor),
			ts.DebugMessage("Should have a runtime attempt, events:\n"+suite.summarizeEvents(executorEvents)))
		suite.Require().Equal(1, countEvents(eventsAfterExecutor, anaConst.CatDebug, anaConst.ActExecutorExit, anaConst.SrcExecutor),
			ts.DebugMessage("Should have an executor exit event, events:\n"+suite.summarizeEvents(executorEvents)))

		// It's possible due to the timing of the heartbeats and the fact that they are async that we have gotten either
		// one or two by this point. Technically more is possible, just very unlikely.
		numHeartbeats := countEvents(executorEvents, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, anaConst.SrcExecutor)
		suite.Require().Greater(numHeartbeats, 0, "Should have a heartbeat")
		suite.Require().LessOrEqual(numHeartbeats, 2, "Should not have excessive heartbeats")
		var heartbeatEvent *reporters.TestLogEntry
		for _, e := range executorEvents {
			if e.Action == anaConst.ActRuntimeHeartbeat {
				heartbeatEvent = &e
			}
		}
		suite.Require().NotNil(heartbeatEvent, "Should have a heartbeat event")
		suite.Require().Equal(*heartbeatEvent.Dimensions.ProjectNameSpace, namespace)
		suite.Require().Equal(*heartbeatEvent.Dimensions.CommitID, commitID)
	})

	/* ACTIVATE SHUTDOWN TESTS */

	cp.SendLine("exit")
	if runtime.GOOS == "windows" {
		// We have to exit twice on windows, as we're running through `cmd /k`
		cp.SendLine("exit")
	}
	suite.Require().NoError(rtutils.Timeout(func() error {
		return cp.ExpectExitCode(0)
	}, 5*time.Second), ts.DebugMessage("Timed out waiting for exit code"))

	time.Sleep(sleepTime) // give time to let rtwatcher detect process has exited

	// Test that we are no longer sending heartbeats

	events = parseAnalyticsEvents(suite, ts)
	suite.Require().NotEmpty(events)
	eventsAfterExit := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, anaConst.SrcExecutor)

	time.Sleep(sleepTime)

	eventsAfter := parseAnalyticsEvents(suite, ts)
	suite.Require().NotEmpty(eventsAfter)
	eventsAfterExitAndWait := countEvents(eventsAfter, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, anaConst.SrcExecutor)

	suite.Equal(eventsAfterExit, eventsAfterExitAndWait,
		fmt.Sprintf("Heartbeats should stop ticking after exiting subshell.\n"+
			"Unexpected events: %s", suite.summarizeEvents(filterHeartbeats(eventsAfter[len(events):])),
		))

	// Ensure any analytics events from the state tool have the instance ID set
	for _, e := range events {
		if strings.Contains(e.Category, "state-svc") || strings.Contains(e.Action, "state-svc") {
			continue
		}
		suite.NotEmpty(e.Dimensions.InstanceID)
	}

	suite.assertSequentialEvents(events)
}

func (suite *AnalyticsIntegrationTestSuite) TestExecEvents() {
	suite.OnlyRunForTags(tagsuite.Analytics, tagsuite.Critical)

	/* TEST SETUP */

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "ActiveState-CLI/Alternate-Python"
	commitID := "efcc851f-1451-4d0a-9dcb-074ac3f35f0a"

	// We want to do a clean test without an activate event, so we have to manually seed the yaml
	ts.PrepareProject(namespace, commitID)

	heartbeatInterval := 1000 // in milliseconds
	sleepTime := time.Duration(heartbeatInterval) * time.Millisecond
	sleepTime = sleepTime + (sleepTime / 2)

	env := []string{
		fmt.Sprintf("%s=%d", constants.HeartbeatIntervalEnvVarName, heartbeatInterval),
	}

	/* EXEC TESTS */

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("exec", "--", "python3", "-c", fmt.Sprintf("import time; time.sleep(%f); print('DONE')", sleepTime.Seconds())),
		e2e.OptWD(ts.Dirs.Work),
		e2e.OptAppendEnv(env...),
	)

	cp.Expect("DONE", e2e.RuntimeSourcingTimeoutOpt)

	time.Sleep(sleepTime)

	events := parseAnalyticsEvents(suite, ts)
	suite.Require().NotEmpty(events)

	runtimeEvents := filterEvents(events, func(e reporters.TestLogEntry) bool {
		return e.Category == anaConst.CatRuntimeUsage
	})

	suite.Equal(1, countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeAttempt, anaConst.SrcStateTool),
		ts.DebugMessage("Should have a runtime attempt, events:\n"+suite.summarizeEvents(runtimeEvents)))

	suite.assertGtEvents(events, 0, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, anaConst.SrcStateTool,
		"Expected new heartbeats after state exec")

	cp.ExpectExitCode(0)
}

func countEvents(events []reporters.TestLogEntry, category, action, source string) int {
	filteredEvents := funk.Filter(events, func(e reporters.TestLogEntry) bool {
		return e.Category == category && e.Action == action && e.Source == source
	}).([]reporters.TestLogEntry)
	return len(filteredEvents)
}

func filterHeartbeats(events []reporters.TestLogEntry) []reporters.TestLogEntry {
	return filterEvents(events, func(e reporters.TestLogEntry) bool {
		return e.Category == anaConst.CatRuntimeUsage && e.Action == anaConst.ActRuntimeHeartbeat
	})
}

func filterEvents(events []reporters.TestLogEntry, filters ...func(e reporters.TestLogEntry) bool) []reporters.TestLogEntry {
	filteredEvents := funk.Filter(events, func(e reporters.TestLogEntry) bool {
		for _, filter := range filters {
			if !filter(e) {
				return false
			}
		}
		return true
	}).([]reporters.TestLogEntry)
	return filteredEvents
}

func (suite *AnalyticsIntegrationTestSuite) assertNEvents(events []reporters.TestLogEntry,
	expectedN int, category, action, source string, errMsg string) {
	suite.Assert().Equal(expectedN, countEvents(events, category, action, source),
		"Expected %d %s:%s events.\nFile location: %s\nEvents received:\n%s\nError:\n%s",
		expectedN, category, action, suite.eventsfile, suite.summarizeEvents(events), errMsg)
}

func (suite *AnalyticsIntegrationTestSuite) assertGtEvents(events []reporters.TestLogEntry,
	greaterThanN int, category, action, source string, errMsg string) {
	suite.Assert().Greater(countEvents(events, category, action, source), greaterThanN,
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
		summary = append(summary, fmt.Sprintf("%s:%s:%s (%s)", event.Category, event.Action, event.Label, event.Source))
	}
	return strings.Join(summary, "\n")
}

func (suite *AnalyticsIntegrationTestSuite) summarizeEventSequence(events []reporters.TestLogEntry) string {
	summary := []string{}
	for _, event := range events {
		summary = append(summary, fmt.Sprintf("%s:%s:%s (%s seq: %s:%s:%d)\n",
			event.Category, event.Action, event.Label, event.Source,
			*event.Dimensions.Command, (*event.Dimensions.InstanceID)[0:6], *event.Dimensions.Sequence))
	}
	return strings.Join(summary, "\n")
}

type TestingSuiteForAnalytics interface {
	Require() *helperSuite.Assertions
}

func parseAnalyticsEvents(suite TestingSuiteForAnalytics, ts *e2e.Session) []reporters.TestLogEntry {
	time.Sleep(time.Second) // give svc time to process events

	file := filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
	suite.Require().FileExists(file, ts.DebugMessage(""))

	b, err := fileutils.ReadFile(file)
	suite.Require().NoError(err)

	var result []reporters.TestLogEntry
	entries := strings.Split(string(b), "\x00")
	for _, entry := range entries {
		if len(entry) == 0 {
			continue
		}

		var parsedEntry reporters.TestLogEntry
		err := json.Unmarshal([]byte(entry), &parsedEntry)
		suite.Require().NoError(err, fmt.Sprintf("path: %s, value: \n%s\n", file, entry))
		result = append(result, parsedEntry)
	}

	return result
}

func (suite *AnalyticsIntegrationTestSuite) TestSend() {
	suite.OnlyRunForTags(tagsuite.Analytics, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	cp := ts.Spawn("--version")
	cp.Expect("Version")
	cp.ExpectExitCode(0)

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	cp = ts.Spawn("config", "set", constants.ReportAnalyticsConfig, "false")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)

	initialEvents := parseAnalyticsEvents(suite, ts)
	suite.assertSequentialEvents(initialEvents)

	cp = ts.Spawn("--version")
	cp.Expect("Version")
	cp.ExpectExitCode(0)

	events := parseAnalyticsEvents(suite, ts)
	currentEvents := len(events)
	if currentEvents > len(initialEvents) {
		suite.Failf("Should not get additional events", "Got %d additional events, should be 0", currentEvents-len(initialEvents))
	}

	suite.assertSequentialEvents(events)
}

func (suite *AnalyticsIntegrationTestSuite) TestSequenceAndFlags() {
	suite.OnlyRunForTags(tagsuite.Analytics)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("--version")
	cp.Expect("Version")
	cp.ExpectExitCode(0)

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
	events := parseAnalyticsEvents(suite, ts)
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

func (suite *AnalyticsIntegrationTestSuite) TestInputError() {
	suite.OnlyRunForTags(tagsuite.Analytics)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	cp := ts.Spawn("clean", "uninstall", "badarg", "--mono")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	events := parseAnalyticsEvents(suite, ts)
	suite.assertSequentialEvents(events)

	suite.assertNEvents(events, 1, anaConst.CatDebug, anaConst.ActCommandInputError, anaConst.SrcStateTool,
		fmt.Sprintf("output:\n%s\n%s",
			cp.Output(), ts.DebugLogsDump()))

	for _, event := range events {
		if event.Category == anaConst.CatDebug && event.Action == anaConst.ActCommandInputError {
			suite.Equal("state clean uninstall --mono", event.Label)
		}
	}
}

func (suite *AnalyticsIntegrationTestSuite) TestAttempts() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Analytics, tagsuite.Debug)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/test", "9090c128-e948-4388-8f7f-96e2c1e00d98")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", "ActiveState-CLI/Alternate-Python"),
		e2e.OptWD(ts.Dirs.Work),
		e2e.OptAppendEnv(constants.DisableActivateEventsEnvVarName+"=false"),
		e2e.OptTermTest(termtest.OptVerboseLogger()),
	)

	cp.Expect("Creating a Virtual Environment")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput()

	cp.SendLine("python3 --version")
	cp.Expect("Python 3.")

	time.Sleep(time.Second) // Ensure state-svc has time to report events

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
	events := parseAnalyticsEvents(suite, ts)

	var foundAttempts int
	var foundExecs int
	for _, e := range events {
		if strings.Contains(e.Category, "runtime") && strings.Contains(e.Action, "attempt") {
			foundAttempts++
			if strings.Contains(*e.Dimensions.Trigger, "exec") && strings.Contains(e.Source, anaConst.SrcExecutor) {
				foundExecs++
			}
		}
	}

	if foundAttempts == 2 {
		suite.Fail("Should find multiple runtime attempts")
	}
	if foundExecs == 1 {
		suite.Fail("Should find one exec event")
	}
}

func (suite *AnalyticsIntegrationTestSuite) TestHeapEvents() {
	suite.OnlyRunForTags(tagsuite.Analytics)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", "ActiveState-CLI/Alternate-Python"),
		e2e.OptWD(ts.Dirs.Work),
	)

	cp.Expect("Creating a Virtual Environment")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput()

	time.Sleep(time.Second) // Ensure state-svc has time to report events

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	events := parseAnalyticsEvents(suite, ts)
	suite.Require().NotEmpty(events)

	// Ensure analytics events have required/important fields
	for _, e := range events {
		// Skip events that are not relevant to Heap
		// State Service, Update, and Auth events can run before a user has logged in
		if strings.Contains(e.Category, "state-svc") || strings.Contains(e.Action, "state-svc") || strings.Contains(e.Action, "auth") || strings.Contains(e.Category, "update") {
			continue
		}

		// UserID is used to identify the user
		suite.NotEmpty(e.Dimensions.UserID, "Event should have a user ID")

		// Category and Action are primary attributes reported to Heap and should be set
		suite.NotEmpty(e.Category, "Event category should not be empty")
		suite.NotEmpty(e.Action, "Event action should not be empty")
	}

	suite.assertSequentialEvents(events)
}

func (suite *AnalyticsIntegrationTestSuite) TestConfigEvents() {
	suite.OnlyRunForTags(tagsuite.Analytics, tagsuite.Config)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("config", "set", "optin.unstable", "false"),
		e2e.OptWD(ts.Dirs.Work),
	)
	cp.Expect("Successfully set config key")

	time.Sleep(time.Second) // Ensure state-svc has time to report events

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("config", "set", "optin.unstable", "true"),
		e2e.OptWD(ts.Dirs.Work),
	)
	cp.Expect("Successfully set config key")

	time.Sleep(time.Second) // Ensure state-svc has time to report events

	suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)

	events := parseAnalyticsEvents(suite, ts)
	suite.Require().NotEmpty(events)

	// Ensure analytics events have required/important fields
	var found int
	for _, e := range events {
		if !strings.Contains(e.Category, anaConst.CatConfig) {
			continue
		}

		if e.Label != "optin.unstable" {
			suite.Fail("Incorrect config event label")
		}
		found++
	}

	if found < 2 {
		suite.Fail("Should find multiple config events")
	}

	suite.assertNEvents(events, 1, anaConst.CatConfig, anaConst.ActConfigSet, anaConst.SrcStateTool, "Should be at one config set event")
	suite.assertNEvents(events, 1, anaConst.CatConfig, anaConst.ActConfigUnset, anaConst.SrcStateTool, "Should be at one config unset event")
	suite.assertSequentialEvents(events)
}

func (suite *AnalyticsIntegrationTestSuite) TestCIAndInteractiveDimensions() {
	suite.OnlyRunForTags(tagsuite.Analytics)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	for _, interactive := range []bool{true, false} {
		suite.T().Run(fmt.Sprintf("interactive: %v", interactive), func(t *testing.T) {
			args := []string{"--version"}
			if !interactive {
				args = append(args, "--non-interactive")
			}
			cp := ts.Spawn(args...)
			cp.Expect("ActiveState CLI")
			cp.ExpectExitCode(0)

			time.Sleep(time.Second) // Ensure state-svc has time to report events

			suite.eventsfile = filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
			events := parseAnalyticsEvents(suite, ts)
			suite.Require().NotEmpty(events)
			processedAnEvent := false
			for _, e := range events {
				if !strings.Contains(e.Category, anaConst.CatRunCmd) || e.Label == "" {
					continue // only look at spawned run-command events
				}
				interactiveEvent := !strings.Contains(e.Label, "--non-interactive")
				if interactive != interactiveEvent {
					continue // ignore the other spawned command
				}
				suite.Equal(condition.OnCI(), *e.Dimensions.CI, "analytics should report being on CI")
				suite.Equal(interactive, *e.Dimensions.Interactive, "analytics did not report the correct interactive value for %v", e)
				suite.Equal(condition.OnCI(), // not InActiveStateCI() because if it's false, we forgot to set ACTIVESTATE_CI env var in GitHub Actions scripts
					*e.Dimensions.ActiveStateCI, "analytics did not report being in ActiveState CI")
				processedAnEvent = true
			}
			suite.True(processedAnEvent, "did not actually test CI and Interactive dimensions")
			suite.assertSequentialEvents(events)
		})
	}
}

func TestAnalyticsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AnalyticsIntegrationTestSuite))
}
