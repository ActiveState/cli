package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type EventsIntegrationTestSuite struct {
	suite.Suite
}

func (suite *EventsIntegrationTestSuite) TestEvents_FirstActivate() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(strings.TrimSpace(`
project: https://platform.activestate.com/ActiveState-CLI/Python3?commitID=fbc613d6-b0b1-4f84-b26e-4aa5869c4e54
events:
  - name: first-activate
    value: echo "First activate event"
  - name: activate
    value: echo "Activate event"
`))

	cp := ts.Spawn("activate")
	cp.Expect("First activate event")
	cp.Expect("Activate event")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("activate")
	cp.Expect("Activate event")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
	output := cp.TrimmedSnapshot()
	if strings.Contains(output, "First activate event") {
		suite.T().Fatal("Output from second activate event should not contain first-activate output")
	}
}

func TestEventsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(EventsIntegrationTestSuite))
}
