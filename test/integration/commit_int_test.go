package integration

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/termtest"
)

type CommitIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CommitIntegrationTestSuite) TestCommitManualBuildScriptMod() {
	suite.OnlyRunForTags(tagsuite.Commit, tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProjectAndBuildScript("ActiveState-CLI/Commit-Test-A", "7a1b416e-c17f-4d4a-9e27-cbad9e8f5655")

	proj, err := project.FromPath(ts.Dirs.Work)
	suite.NoError(err, "Error loading project")

	_, err = buildscript_runbit.ScriptFromProject(proj)
	suite.Require().NoError(err) // verify validity

	cp := ts.Spawn("commit")
	cp.Expect("no new changes")
	cp.ExpectExitCode(0)

	_, err = buildscript_runbit.ScriptFromProject(proj)
	suite.Require().NoError(err) // verify validity

	scriptPath := filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName)
	data := fileutils.ReadFileUnsafe(scriptPath)
	data = bytes.ReplaceAll(data, []byte("casestyle"), []byte("case"))
	suite.Require().NoError(fileutils.WriteFile(scriptPath, data), "Update buildscript")

	ts.LoginAsPersistentUser() // for CVE reporting

	cp = ts.Spawn("commit")
	cp.Expect("Operating on project")
	cp.Expect("Creating commit")
	cp.Expect("Resolving Dependencies")
	cp.Expect("Installing case@")
	cp.Expect("Checking for vulnerabilities")
	cp.Expect("successfully created")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("pkg")
	cp.Expect("case ", e2e.RuntimeSourcingTimeoutOpt) // note: intentional trailing whitespace to not match 'casestyle'
	cp.ExpectExitCode(0)
}

func (suite *CommitIntegrationTestSuite) TestCommitAtTimeChange() {
	suite.OnlyRunForTags(tagsuite.Commit, tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProjectAndBuildScript("ActiveState-CLI/Commit-Test-A", "7a1b416e-c17f-4d4a-9e27-cbad9e8f5655")

	proj, err := project.FromPath(ts.Dirs.Work)
	suite.NoError(err, "Error loading project")

	_, err = buildscript_runbit.ScriptFromProject(proj)
	suite.Require().NoError(err) // verify validity

	// Update top-level at_time variable.
	dateTime := "2023-06-21T12:34:56Z"
	buildScriptFile := filepath.Join(proj.Dir(), constants.BuildScriptFileName)
	contents, err := fileutils.ReadFile(buildScriptFile)
	suite.Require().NoError(err)
	contents = bytes.Replace(contents, []byte("2023-06-22T21:56:10Z"), []byte(dateTime), 1)
	suite.Require().NoError(fileutils.WriteFile(buildScriptFile, contents))
	suite.Require().Contains(string(fileutils.ReadFileUnsafe(filepath.Join(proj.Dir(), constants.BuildScriptFileName))), dateTime)

	cp := ts.Spawn("commit")
	cp.Expect("successfully created", termtest.OptExpectErrorMessage(string(contents)))
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.Expect("Revision")
	revisionTime, err := time.Parse(time.RFC3339, dateTime)
	suite.Require().NoError(err)
	cp.Expect(revisionTime.Format(time.RFC822))
	cp.ExpectExitCode(0)
}

func TestCommitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CommitIntegrationTestSuite))
}
