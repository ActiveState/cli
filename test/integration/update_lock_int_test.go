package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
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

	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(cfg)

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("update", "lock"),
		e2e.OptAppendEnv(suite.env(false, false)...),
	)

	cp.Expect("Version locked at")
	cp.ExpectExitCode(0)

	suite.versionCompare(ts, constants.Version, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestLockedChannel() {
	suite.OnlyRunForTags(tagsuite.Update)
	targetBranch := "release"
	if constants.BranchName == "release" {
		targetBranch = "master"
	}
	tests := []struct {
		name            string
		lock            string
		expectLockError bool
		expectedChannel string
	}{
		{
			"oldVersion",
			oldUpdateVersion,
			true,
			"beta",
		},
		{
			"channel",
			targetBranch,
			true,
			targetBranch,
		},
		{
			"locked-multi-file-version",
			fmt.Sprintf("%s@0.29.0-SHA000000", targetBranch),
			true,
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

			cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
			suite.Require().NoError(err)
			defer cfg.Close()

			yamlPath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
			pjfile.SetPath(yamlPath)
			pjfile.Save(cfg)

			cp := ts.SpawnWithOpts(
				e2e.OptArgs("update", "lock", "--set-channel", tt.lock),
				e2e.OptAppendEnv(suite.env(false, false)...),
			)
			cp.Expect("Version locked at")
			cp.Expect(tt.expectedChannel + "@")
			cp.ExpectExitCode(0)

			yamlContents, err := fileutils.ReadFile(yamlPath)
			suite.Require().NoError(err)
			suite.Contains(string(yamlContents), tt.lock)

			if tt.expectLockError {
				cp = ts.SpawnWithOpts(e2e.OptArgs("--version"), e2e.OptAppendEnv(suite.env(true, false)...))
				cp.Expect("This project is locked at State Tool version")
				cp.ExpectExitCode(1)
				return
			}
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

			cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
			suite.Require().NoError(err)
			defer cfg.Close()

			pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
			pjfile.Save(cfg)

			args := []string{"update", "lock"}
			if tt.Forced {
				args = append(args, "--non-interactive")
			}
			cp := ts.SpawnWithOpts(
				e2e.OptArgs(args...),
				e2e.OptAppendEnv(suite.env(true, true)...),
			)
			cp.Expect("sure you want")
			if tt.Confirm || tt.Forced {
				cp.Send("y")
				cp.Expect("Version locked at")
			} else {
				cp.Send("n")
				cp.Expect("Cancelling")
			}
			cp.ExpectNotExitCode(0)
		})
	}
}

func (suite *UpdateIntegrationTestSuite) TestLockUnlock() {
	suite.OnlyRunForTags(tagsuite.Update)

	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cfg, err := config.New()
	suite.Require().NoError(err)
	defer cfg.Close()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(cfg)

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("update", "lock", "--non-interactive"),
		e2e.OptAppendEnv(suite.env(false, false)...),
	)
	cp.Expect("locked at")

	data, err := ioutil.ReadFile(pjfile.Path())
	suite.Require().NoError(err)

	lockRegex := regexp.MustCompile(`(?m)^lock:.*`)
	suite.Assert().True(lockRegex.Match(data), "lock info was not written to "+pjfile.Path())

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("update", "unlock", "-n"),
		e2e.OptAppendEnv(suite.env(false, false)...),
	)
	cp.Expect("unlocked")

	data, err = ioutil.ReadFile(pjfile.Path())
	suite.Require().NoError(err)
	suite.Assert().False(lockRegex.Match(data), "lock info was not removed from "+pjfile.Path())
}

func (suite *UpdateIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("update", "lock", "-o", "json")
	cp.Expect(`"channel":`)
	cp.Expect(`"version":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}
