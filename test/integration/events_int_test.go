package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type EventsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *EventsIntegrationTestSuite) TestEvents() {
	suite.OnlyRunForTags(tagsuite.Events)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(strings.TrimSpace(`
project: https://platform.activestate.com/ActiveState-CLI/Python3
scripts:
  - name: before
    language: bash
    value: echo before-script
  - name: after
    language: bash
    value: echo after-script
events:
  - name: first-activate
    value: echo "First activate event"
  - name: activate
    value: echo "Activate event"
  - name: activate
    value: echo "Activate event duplicate"
  - name: before-command
    scope: ["activate"]
    value: before
  - name: after-command
    scope: ["activate"]
    value: after
`))
	ts.PrepareCommitIdFile("fbc613d6-b0b1-4f84-b26e-4aa5869c4e54")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate"),
		e2e.OptAppendEnv(constants.DisableActivateEventsEnvVarName+"=false"),
	)
	cp.SendEnter()
	cp.Expect("before-script")
	cp.Expect("First activate event")
	cp.Expect("Activate event")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.Expect("after-script")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("activate"),
		e2e.OptAppendEnv(constants.DisableActivateEventsEnvVarName+"=false"),
	)
	cp.Expect("Activate event")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
	output := cp.Output()
	if strings.Contains(output, "First activate event") {
		suite.T().Fatal("Output from second activate event should not contain first-activate output")
	}
	if strings.Contains(output, "Activate event duplicate") {
		suite.T().Fatal("Output should not contain output from duplicate activate event")
	}
}

func (suite *EventsIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Events, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("events", "-o", "json")
	cp.Expect(`[{"event":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestEventsIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(EventsIntegrationTestSuite))
}
