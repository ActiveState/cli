package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type NotificationIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *NotificationIntegrationTestSuite) TestNotification_None() {
	suite.OnlyRunForTags(tagsuite.Notifications, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("--version")
	cp.Expect("ActiveState CLI by ActiveState Software Inc.")
	cp.ExpectExitCode(0)

	// Note: since message failures should fail silently without impacting the user we need to check
	// the logs for any potential issues. This is done automatically by ts.Close().
}

func (suite *NotificationIntegrationTestSuite) TestNotification_Basic() {
	suite.OnlyRunForTags(tagsuite.Notifications, tagsuite.Critical)
	tests := []struct {
		Name         string
		MessageJson  string
		ExpectRepeat bool
		ExpectShown  bool
	}{
		{
			"Defaults",
			`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification"
			}]`,
			false,
			true,
		},
		{
			"Repeat Hourly",
			`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"Repeat": "Hourly"
			}]`,
			false,
			true,
		},
		{
			"Repeat Constantly",
			`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"Repeat": "Constantly"
			}]`,
			true,
			true,
		},
		{
			"Within Date Range",
			fmt.Sprintf(`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"StartDate": "%s",
				"EndDate": "%s"
			}]`,
				time.Now().Add(-1*time.Hour).Format(time.RFC3339),
				time.Now().Add(1*time.Hour).Format(time.RFC3339)),
			false,
			true,
		},
		{
			"Outside Date Range",
			fmt.Sprintf(`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"StartDate": "%s",
				"EndDate": "%s"
			}]`,
				time.Now().Add(1*time.Hour).Format(time.RFC3339),
				time.Now().Add(2*time.Hour).Format(time.RFC3339)),
			false,
			false,
		},
		{
			"Only Start Date - Inside Range",
			fmt.Sprintf(`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"StartDate": "%s"
			}]`,
				time.Now().Add(-1*time.Hour).Format(time.RFC3339)),
			false,
			true,
		},
		{
			"Only End Date - Inside Range",
			fmt.Sprintf(`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"EndDate": "%s"
			}]`,
				time.Now().Add(1*time.Hour).Format(time.RFC3339)),
			false,
			true,
		},
		{
			"Outside Date Range - Future",
			fmt.Sprintf(`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"StartDate": "%s",
				"EndDate": "%s"
			}]`,
				time.Now().Add(1*time.Hour).Format(time.RFC3339),
				time.Now().Add(2*time.Hour).Format(time.RFC3339)),
			false,
			false,
		},
		{
			"Outside Date Range - Past",
			fmt.Sprintf(`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"StartDate": "%s",
				"EndDate": "%s"
			}]`,
				time.Now().Add(-2*time.Hour).Format(time.RFC3339),
				time.Now().Add(-1*time.Hour).Format(time.RFC3339)),
			false,
			false,
		},
		{
			"Only Start Date - Outside Range",
			fmt.Sprintf(`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"StartDate": "%s"
			}]`,
				time.Now().Add(1*time.Hour).Format(time.RFC3339)),
			false,
			false,
		},
		{
			"Only End Date - Outside Range",
			fmt.Sprintf(`[{
				"ID": "simple",
				"Message": "This is a [NOTICE]simple[/RESET] notification",
				"EndDate": "%s"
			}]`,
				time.Now().Add(-1*time.Hour).Format(time.RFC3339)),
			false,
			false,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(tt.MessageJson), 0755)
			suite.Require().NoError(err)

			cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(constants.NotificationsOverrideEnvVarName+"="+msgFile))

			if !tt.ExpectShown {
				suite.Require().NotContains(cp.Output(), "This is a simple notification", "Notification should not appear when outside date range")
			} else {
				cp.Expect(`This is a simple notification`)
			}

			cp.Expect("ActiveState CLI by ActiveState Software Inc.")
			cp.ExpectExitCode(0)

			// Ensure message doesn't stick around when we run another command
			cp = ts.Spawn("--version")
			if tt.ExpectRepeat {
				cp.Expect(`This is a simple notification`)
			}
			cp.ExpectExitCode(0)
			if !tt.ExpectRepeat {
				suite.Require().NotContains(cp.Output(), "This is a simple notification", "Should not repeat as that's the default behavior")
			}
		})
	}
}

func (suite *NotificationIntegrationTestSuite) TestNotification_Basic_PlacementAfter() {
	suite.OnlyRunForTags(tagsuite.Notifications)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(fmt.Sprintf(`[
	{
		"ID": "simple",
		"Message": "This is a [NOTICE]simple[/RESET] notification",
		"Placement": "%s"
	}
]`, graph.NotificationPlacementTypeAfterCmd)), 0755)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(constants.NotificationsOverrideEnvVarName+"="+msgFile))
	cp.Expect("ActiveState CLI by ActiveState Software Inc.")
	cp.Expect(`This is a simple notification`)
	cp.ExpectExitCode(0)
}

func (suite *NotificationIntegrationTestSuite) TestNotification_Basic_InterruptPrompt() {
	suite.OnlyRunForTags(tagsuite.Notifications)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(fmt.Sprintf(`[
	{
		"ID": "simple",
		"Message": "This is a [NOTICE]simple[/RESET] notification",
		"Repeat": "Constantly",
		"Interrupt": "%s"
	}
]`, graph.NotificationInterruptTypePrompt)), 0755)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(constants.NotificationsOverrideEnvVarName+"="+msgFile))
	cp.Expect(`This is a simple notification`)
	cp.Expect("Press ENTER to continue")
	time.Sleep(time.Millisecond * 100)
	suite.Require().NotContains(cp.Output(), "ActiveState CLI by ActiveState Software Inc.")
	cp.SendEnter()
	cp.Expect("ActiveState CLI by ActiveState Software Inc.")
	cp.ExpectExitCode(0)

	// Test that non-interactive does not prompt
	cp = ts.SpawnWithOpts(e2e.OptArgs("--version", "-n"), e2e.OptAppendEnv(constants.NotificationsOverrideEnvVarName+"="+msgFile))
	cp.Expect(`This is a simple notification`)
	cp.Expect("ActiveState CLI by ActiveState Software Inc.")
	cp.ExpectExitCode(0)
	suite.Require().NotContains(cp.Output(), "Press ENTER to continue")
}

func (suite *NotificationIntegrationTestSuite) TestNotification_Basic_InterruptExit() {
	suite.OnlyRunForTags(tagsuite.Notifications)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(fmt.Sprintf(`[
	{
		"ID": "simple",
		"Message": "This is a [NOTICE]simple[/RESET] notification",
		"Interrupt": "%s"
	}
]`, graph.NotificationInterruptTypeExit)), 0755)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(constants.NotificationsOverrideEnvVarName+"="+msgFile))
	cp.ExpectExitCode(1)
	suite.Require().Contains(cp.Snapshot(), "This is a simple notification")
	suite.Require().NotContains(cp.Output(), "ActiveState CLI by ActiveState Software Inc.")
	ts.IgnoreLogErrors()
}

func (suite *NotificationIntegrationTestSuite) TestNotificationEndpoint_SetBeforeInvocation() {
	suite.OnlyRunForTags(tagsuite.Notifications)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	notificationsURL := "https://test.example.com/notifications"
	ts.SetConfig(constants.NotificationsURLConfig, notificationsURL)
	suite.Assert().Equal(ts.GetConfig(constants.NotificationsURLConfig), notificationsURL)

	cp := ts.Spawn("--version")
	cp.ExpectExitCode(0)

	foundConfigURL := false
	foundDefaultURL := false
	logDir := filepath.Join(ts.Dirs.Config, "logs")
	files, err := os.ReadDir(logDir)
	suite.Require().NoError(err)
	for _, file := range files {
		if strings.Contains(file.Name(), "state-svc") {
			logPath := filepath.Join(logDir, file.Name())
			contents := string(fileutils.ReadFileUnsafe(logPath))
			if strings.Contains(contents, "test.example.com") {
				foundConfigURL = true
			}
			if strings.Contains(contents, "state-tool.s3.amazonaws.com") {
				foundDefaultURL = true
			}
		}
	}
	suite.Assert().True(foundConfigURL, "Log file should contain the configured notifications endpoint '%s'", notificationsURL)
	suite.Assert().False(foundDefaultURL, "Log file should not contain the default notifications endpoint '%s'", constants.NotificationsInfoURL)

	// Clean up - remove the config setting
	cp = ts.Spawn("config", "set", constants.NotificationsURLConfig, "")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)
}

func (suite *NotificationIntegrationTestSuite) TestNotificationEndpoint() {
	suite.OnlyRunForTags(tagsuite.Notifications)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	msgFile, err := fileutils.WriteTempFileToDir(ts.Dirs.Work, "messages.json", []byte(`[{
		"ID": "config-test",
		"Message": "This is a [NOTICE]config test[/RESET] notification"
	}]`), 0755)
	suite.Require().NoError(err)

	notificationsURL := "https://test.example.com/notifications"
	cp := ts.Spawn("config", "set", constants.NotificationsURLConfig, notificationsURL)
	cp.Expect("Successfully set config key")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("--version"),
		e2e.OptAppendEnv(constants.NotificationsOverrideEnvVarName+"="+msgFile),
		e2e.OptAppendEnv("VERBOSE=true"),
	)
	cp.ExpectExitCode(0)

	correctHostFound := false
	incorrectHostFound := false
	for _, path := range ts.LogFiles() {
		contents := string(fileutils.ReadFileUnsafe(path))
		if strings.Contains(contents, notificationsURL) {
			correctHostFound = true
			break
		}
		if strings.Contains(contents, constants.NotificationsInfoURL) {
			incorrectHostFound = true
			break
		}
	}
	suite.Assert().True(correctHostFound, "Log file should contain the configured notifications endpoint 'example.com'")
	suite.Assert().False(incorrectHostFound, "Log file should not contain the default notifications endpoint 'state-tool.s3.amazonaws.com'")

	// Clean up - remove the config setting
	cp = ts.Spawn("config", "set", constants.NotificationsURLConfig, "")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)
}

func TestNotificationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationIntegrationTestSuite))
}
