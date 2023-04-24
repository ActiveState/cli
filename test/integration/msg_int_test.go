package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type MsgIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *MsgIntegrationTestSuite) TestMessage_Basic_Defaults() {
	suite.OnlyRunForTags(tagsuite.Messaging, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(`[
	{
		"ID": "simple",
		"Message": "This is a [NOTICE]simple[/RESET] message"
	}
]`), 0755)
	suite.Require().NoError(err)

	// We test on config as it just dumps help and has minimal output
	// The base state command would also work, but it's output is more verbose and termtest likes to cut off content if it's too long
	cp := ts.SpawnWithOpts(e2e.WithArgs("config"), e2e.AppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.Expect(`This is a simple message`)
	cp.Expect("Usage:")
	cp.ExpectExitCode(0)

	// Ensure message doesn't stick around when we run another command
	cp = ts.Spawn()
	cp.ExpectExitCode(0)
	suite.Require().NotContains(cp.Snapshot(), "This is a simple message", "Should not repeat as that's the default behavior")
}

func (suite *MsgIntegrationTestSuite) TestMessage_Basic_PlacementAfter() {
	suite.OnlyRunForTags(tagsuite.Messaging)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(fmt.Sprintf(`[
	{
		"ID": "simple",
		"Message": "This is a [NOTICE]simple[/RESET] message",
		"Placement": "%s"
	}
]`, graph.MessagePlacementTypeAfterCmd)), 0755)
	suite.Require().NoError(err)

	// We test on config as it just dumps help and has minimal output
	// The base state command would also work, but it's output is more verbose and termtest likes to cut off content if it's too long
	cp := ts.SpawnWithOpts(e2e.WithArgs("config"), e2e.AppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.Expect("Usage:")
	cp.Expect(`This is a simple message`)
	cp.ExpectExitCode(0)
}

func (suite *MsgIntegrationTestSuite) TestMessage_Basic_InterruptPrompt() {
	suite.OnlyRunForTags(tagsuite.Messaging)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(fmt.Sprintf(`[
	{
		"ID": "simple",
		"Message": "This is a [NOTICE]simple[/RESET] message",
		"Interrupt": "%s"
	}
]`, graph.MessageInterruptTypePrompt)), 0755)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.WithArgs("config"), e2e.AppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.Expect(`This is a simple message`)
	cp.Expect("Press ENTER to continue")
	time.Sleep(time.Millisecond * 100)
	suite.Require().NotContains(cp.Snapshot(), "Usage:")
	cp.SendLine("")
	cp.Expect("Usage:")
	cp.ExpectExitCode(0)

	// Test that non-interactive does not prompt
	cp = ts.SpawnWithOpts(e2e.WithArgs("config", "-n"), e2e.AppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.Expect(`This is a simple message`)
	cp.Expect("Usage:")
	cp.ExpectExitCode(0)
	suite.Require().NotContains(cp.Snapshot(), "Press ENTER to continue")
}

func (suite *MsgIntegrationTestSuite) TestMessage_Basic_InterruptExit() {
	suite.OnlyRunForTags(tagsuite.Messaging)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(fmt.Sprintf(`[
	{
		"ID": "simple",
		"Message": "This is a [NOTICE]simple[/RESET] message",
		"Interrupt": "%s"
	}
]`, graph.MessageInterruptTypeExit)), 0755)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.WithArgs("config"), e2e.AppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.Expect(`This is a simple message`)
	cp.ExpectExitCode(1)
	suite.Require().NotContains(cp.Snapshot(), "Usage:")
}

func TestMsgIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(MsgIntegrationTestSuite))
}
