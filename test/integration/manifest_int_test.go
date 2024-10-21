package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ManifestIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ManifestIntegrationTestSuite) TestManifest() {
	suite.OnlyRunForTags(tagsuite.Manifest, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("checkout", "ActiveState/cli#9eee7512-b2ab-4600-b78b-ab0cf2e817d8", ".")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.Spawn("manifest")
	cp.Expect("Operating on project: ActiveState/cli")
	cp.Expect("Name")
	cp.Expect("python")
	cp.Expect("3.9.13")
	cp.Expect("1 Critical,")
	cp.Expect("psutil")
	cp.Expect("auto → 5.9.0")
	cp.Expect("None detected")
	cp.ExpectExitCode(0)

	// Ensure that `state manifest` utilized the cache (checkout should've warmed it)
	logFile := ts.LogFiles()[0]
	log := string(fileutils.ReadFileUnsafe(logFile))
	matched := false
	for _, line := range strings.Split(log, "\n") {
		if strings.Contains(line, "GetCache FetchCommit-") {
			suite.Require().Regexp(regexp.MustCompile(`FetchCommit-.*result size: [1-9]`), line)
			matched = true
			break
		}
	}
	suite.Require().True(matched, "log file should contain a line with the FetchCommit call", log)
}

func (suite *ManifestIntegrationTestSuite) TestManifest_JSON() {
	suite.OnlyRunForTags(tagsuite.Manifest)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("checkout", "ActiveState/cli#9eee7512-b2ab-4600-b78b-ab0cf2e817d8", ".")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.Spawn("manifest", "--output", "json")
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
	cp.Expect(`"requirements":`)
}

func (suite *ManifestIntegrationTestSuite) TestManifest_Advanced_Reqs() {
	suite.OnlyRunForTags(tagsuite.Manifest, tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("config", "set", constants.OptinBuildscriptsConfig, "true")
	cp.ExpectExitCode(0)

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI-Testing/Python-With-Custom-Reqs?branch=main&commitID=92ac7df2-0b0c-42f5-9b25-75b0cb4063f7
config_version: 1`) // need config_version to be 1 or more so the migrator does not wipe out our build script
	bsf := filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName)
	err := fileutils.WriteFile(bsf, []byte(fmt.Sprintf(
		"```\n"+
			"Project: https://platform.activestate.com/ActiveState-CLI-Testing/Python-With-Custom-Reqs?branch=main&commitID=92ac7df2-0b0c-42f5-9b25-75b0cb4063f7\n"+
			"Time: 2022-07-07T19:51:01.140Z\n"+
			"```\n"+`
runtime = state_tool_artifacts_v1(src = sources)
sources = solve(
	at_time = TIME,
	requirements = [
		Req(name = "python", namespace = "language", version = Eq(value = "3.9.13")),
		Revision(name = "IngWithRevision", revision_id = "%s"),
		Unrecognized(name = "SomeOpt", value = "SomeValue")
	]
)
main = runtime
`, e2e.CommitIDNotChecked)))
	suite.Require().NoError(err)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("manifest"),
		e2e.OptAppendEnv(constants.DisableBuildscriptDirtyCheck+"=true"), // Don't want to commit buildscript
	)
	cp.ExpectRe(`IngWithRevision\s+` + e2e.CommitIDNotChecked[0:8] + " ")
	cp.Expect("WARNING")
	cp.Expect("project has additional build criteria")
	cp.Expect("Unrecognized")
	cp.Expect(`name = "SomeOpt", value = "SomeValue"`)
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("manifest", "--output", "json"),
		e2e.OptAppendEnv(constants.DisableBuildscriptDirtyCheck+"=true"), // Don't want to commit buildscript
	)
	cp.ExpectExitCode(0)
	out := cp.Output()
	out = strings.Replace(out, "\n", "", -1) // Work around words being wrapped on Windows
	suite.Require().Contains(out, `{"name":"IngWithRevision","version":{"requested":"00000000-0000-0000-0000-000000000000","resolved":"00000000-0000-0000-0000-000000000000"}}`)
	suite.Require().Contains(out, `"unknown_requirements":[{"name":"Unrecognized","value":"name = \"SomeOpt\", value = \"SomeValue\""}]`)
}

func TestManifestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ManifestIntegrationTestSuite))
}
