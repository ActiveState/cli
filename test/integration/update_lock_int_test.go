package integration

import (
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func (suite *UpdateIntegrationTestSuite) TestLocked() {
	suite.OnlyRunForTags(tagsuite.Update)
	suite.T().Skip("Requires https://www.pivotaltracker.com/story/show/177827538 and needs to be adapted.")
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(suite.cfg)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "lock"),
		e2e.AppendEnv(suite.env(false, false)...),
	)

	cp.Expect("Version locked at")
	cp.ExpectExitCode(0)

	suite.versionCompare(ts, false, false, constants.Version, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestLockedChannel() {
	suite.OnlyRunForTags(tagsuite.Update)
	tests := []struct {
		name            string
		lock            string
		expectedChannel string
	}{
		{
			"oldVersion",
			oldUpdateVersion,
			"beta",
		},
		{
			"channel",
			targetBranch,
			targetBranch,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			pjfile := projectfile.Project{
				Project: lockedProjectURL(),
			}
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			// Ensure we always use a unique exe for updates
			ts.UseDistinctStateExes()

			yamlPath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
			pjfile.SetPath(yamlPath)
			pjfile.Save(suite.cfg)

			cp := ts.SpawnWithOpts(
				e2e.WithArgs("update", "lock", "--set-channel", tt.lock),
				e2e.AppendEnv(suite.env(false, false)...),
			)

			cp.Expect("Version locked at")
			cp.Expect(tt.expectedChannel + "@")
			cp.ExpectExitCode(0)

			yamlContents, err := fileutils.ReadFile(yamlPath)
			suite.Require().NoError(err)
			suite.Contains(string(yamlContents), tt.lock)

			suite.branchCompare(ts, false, false, tt.expectedChannel, suite.Equal)
		})
	}
}

func (suite *UpdateIntegrationTestSuite) TestUpdateLockedConfirmation() {
	tests := []struct {
		Name    string
		Confirm bool
		Forced  bool
	}{
		{"Negative", false, false},
		{"Positive", true, false},
		{"Forced", true, true},
	}

	for _, tt := range tests {
		if tt.Forced || tt.Confirm {
			suite.T().Skip("Requires https://www.pivotaltracker.com/story/show/177827538 and needs to be adapted.")
		}
		suite.Run(tt.Name, func() {
			suite.OnlyRunForTags(tagsuite.Update)
			pjfile := projectfile.Project{
				Project: lockedProjectURL(),
				Lock:    fmt.Sprintf("%s@%s", constants.BranchName, constants.Version),
			}

			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			// Ensure we always use a unique exe for updates
			ts.UseDistinctStateExes()

			pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
			pjfile.Save(suite.cfg)

			args := []string{"update", "lock"}
			if tt.Forced {
				args = append(args, "--force")
			}
			cp := ts.SpawnWithOpts(
				e2e.WithArgs(args...),
				e2e.AppendEnv(suite.env(true, true)...),
			)
			cp.Expect("sure you want")
			if tt.Confirm || tt.Forced {
				cp.Send("y")
				cp.Expect("Version locked at")
			} else {
				cp.Send("n")
				cp.Expect("not confirm")
			}
			cp.ExpectNotExitCode(0)
		})
	}
}
