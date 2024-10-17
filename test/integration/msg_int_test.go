package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type MsgIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *MsgIntegrationTestSuite) TestMessage_None() {
	suite.OnlyRunForTags(tagsuite.Messaging, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("--version")
	cp.Expect("ActiveState CLI by ActiveState Software Inc.")
	cp.ExpectExitCode(0)

	// Note: since message failures should fail silently without impacting the user we need to check
	// the logs for any potential issues. This is done automatically by ts.Close().
}

func (suite *MsgIntegrationTestSuite) TestMessage_Basic() {
	suite.OnlyRunForTags(tagsuite.Messaging, tagsuite.Critical)
	tests := []struct {
		Name         string
		MessageJson  string
		ExpectRepeat bool
	}{
		{
			"Defaults",
			`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] message"
			}]`,
			false,
		},
		{
			"Repeat Hourly",
			`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] message",
				"Repeat": "Hourly"
			}]`,
			false,
		},
		{
			"Repeat Constantly",
			`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] message",
				"Repeat": "Constantly"
			}]`,
			true,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(tt.MessageJson), 0755)
			suite.Require().NoError(err)

			cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
			cp.Expect(`This is a simple message`)
			cp.Expect("ActiveState CLI by ActiveState Software Inc.")
			cp.ExpectExitCode(0)

			// Ensure message doesn't stick around when we run another command
			cp = ts.Spawn("--version")
			if tt.ExpectRepeat {
				cp.Expect(`This is a simple message`)
			}
			cp.ExpectExitCode(0)
			if !tt.ExpectRepeat {
				suite.Require().NotContains(cp.Output(), "This is a simple message", "Should not repeat as that's the default behavior")
			}
		})
	}
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

	cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.Expect("ActiveState CLI by ActiveState Software Inc.")
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
		"Repeat": "Constantly",
		"Interrupt": "%s"
	}
]`, graph.MessageInterruptTypePrompt)), 0755)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.Expect(`This is a simple message`)
	cp.Expect("Press ENTER to continue")
	time.Sleep(time.Millisecond * 100)
	suite.Require().NotContains(cp.Output(), "ActiveState CLI by ActiveState Software Inc.")
	cp.SendEnter()
	cp.Expect("ActiveState CLI by ActiveState Software Inc.")
	cp.ExpectExitCode(0)

	// Test that non-interactive does not prompt
	cp = ts.SpawnCmdWithOpts("state", e2e.OptArgs("--version", "-n"), e2e.OptAppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.Expect(`This is a simple message`)
	cp.Expect("ActiveState CLI by ActiveState Software Inc.")
	cp.ExpectExitCode(0)
	suite.Require().NotContains(cp.Output(), "Press ENTER to continue")
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

	cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(constants.MessagesOverrideEnvVarName+"="+msgFile))
	cp.ExpectExitCode(1)
	suite.Require().Contains(cp.Snapshot(), "This is a simple message")
	suite.Require().NotContains(cp.Output(), "ActiveState CLI by ActiveState Software Inc.")
	ts.IgnoreLogErrors()
}

func TestMsgIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(MsgIntegrationTestSuite))
}
