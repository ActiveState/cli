package integration

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/builds"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type BuildsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *BuildsIntegrationTestSuite) TestBuilds() {
	suite.OnlyRunForTags(tagsuite.Builds)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Python-With-Custom-Builds", "993454c7-6613-4b1a-8981-1cee43cc249e")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	// In order to reuse the runtime cache and reduce test time we run the rest in subtests

	suite.Run("no flags", func() {
		cp = ts.Spawn("builds")
		cp.Expect("Operating on project ActiveState-CLI/Python-With-Custom-Builds, located at")
		cp.Expect("CentOS")
		cp.Expect("Docker Image")
		cp.Expect("Installer")
		cp.Expect("macOS")
		cp.Expect("No builds")
		cp.Expect("Windows")
		cp.Expect("Signed Installer")
		cp.Expect(".exe")
		cp.Expect("To download builds run")
		cp.ExpectExitCode(0)
	})

	suite.Run("--all flag", func() {
		cp = ts.Spawn("builds", "--all")
		cp.Expect("CentOS")
		cp.Expect("Docker Image")
		cp.Expect("Installer")
		cp.Expect("Packages")
		cp.Expect("python@3")
		cp.Expect("macOS")
		cp.Expect("Windows")
		cp.Expect("Signed Installer")
		cp.Expect("Packages")
		cp.Expect("python@3")
		cp.Expect("To download builds run")
		cp.ExpectExitCode(0)
	})

	suite.Run("json without flags", func() {
		cp = ts.SpawnWithOpts(e2e.OptArgs("builds", "--output=json"), e2e.OptTermTest(termtest.OptRows(100)))
		cp.ExpectExitCode(0)

		output := builds.StructuredOutput{}
		out := strings.TrimLeft(cp.StrippedSnapshot(), locale.T("notice_runtime_disabled"))
		suite.Require().NoError(json.Unmarshal([]byte(out), &output), ts.DebugMessage(""))

		suite.Equal(3, len(output.Platforms))
		for _, platform := range output.Platforms {
			if !strings.HasPrefix(platform.Name, "macOS") {
				suite.Greater(len(platform.Builds), 0)
			}
			suite.Equal(0, len(platform.Packages))
		}
	})

	suite.Run("json with --all flag", func() {
		cp = ts.SpawnWithOpts(e2e.OptArgs("builds", "--output=json", "--all"), e2e.OptTermTest(termtest.OptRows(100)))
		cp.ExpectExitCode(0)

		output := builds.StructuredOutput{}
		out := strings.TrimLeft(cp.StrippedSnapshot(), locale.T("notice_runtime_disabled"))
		suite.Require().NoError(json.Unmarshal([]byte(out), &output), ts.DebugMessage(""))

		suite.Equal(3, len(output.Platforms))
		for _, platform := range output.Platforms {
			if !strings.HasPrefix(platform.Name, "macOS") {
				suite.Greater(len(platform.Builds), 0)
			}
			suite.Greater(len(platform.Packages), 0)
		}
	})
}

func (suite *BuildsIntegrationTestSuite) TestBuilds_Remote() {
	suite.OnlyRunForTags(tagsuite.Builds)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.Run("Namespace only", func() {
		cp := ts.Spawn("builds", "--namespace", "ActiveState-CLI/Python-With-Custom-Builds")
		cp.Expect("CentOS")
		cp.Expect("Docker Image")
		cp.Expect("Installer")
		cp.Expect("macOS")
		cp.Expect("No builds")
		cp.Expect("Windows")
		cp.Expect("Signed Installer")
		cp.Expect(".exe")
		cp.Expect("To download builds run")
		suite.Assert().NotContains(cp.Snapshot(), "Operating on project")
		cp.ExpectExitCode(0)
	})

	suite.Run("Namespace and commit ID", func() {
		cp := ts.Spawn("builds", "--namespace", "ActiveState-CLI/Python-With-Custom-Builds", "--commit", "993454c7-6613-4b1a-8981-1cee43cc249e")
		cp.Expect("CentOS")
		cp.Expect("Docker Image")
		cp.Expect("Installer")
		cp.Expect("macOS")
		cp.Expect("No builds")
		cp.Expect("Windows")
		cp.Expect("Signed Installer")
		cp.Expect(".exe")
		cp.Expect("To download builds run")
		cp.ExpectExitCode(0)
	})
}

func (suite *BuildsIntegrationTestSuite) TestBuilds_Download() {
	suite.OnlyRunForTags(tagsuite.Builds)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Python-With-Custom-Builds", "993454c7-6613-4b1a-8981-1cee43cc249e")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("builds", "dl", "a46a74e9", "."),
	)
	cp.Expect("Operating on project ActiveState-CLI/Python-With-Custom-Builds, located at")
	cp.Expect("Downloaded bzip2", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
	require.FileExists(suite.T(), filepath.Join(ts.Dirs.Work, "artifact.tar.gz"))
}

func (suite *BuildsIntegrationTestSuite) TestBuilds_Download_Remote() {
	suite.OnlyRunForTags(tagsuite.Builds)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("builds", "dl", "a46a74e9", ".", "--namespace", "ActiveState-CLI/Python-With-Custom-Builds")
	cp.Expect("Downloaded bzip2", e2e.RuntimeSourcingTimeoutOpt)
	suite.Assert().NotContains(cp.Snapshot(), "Operating on project")
	cp.ExpectExitCode(0)
	require.FileExists(suite.T(), filepath.Join(ts.Dirs.Work, "artifact.tar.gz"))
}

func TestBuildsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BuildsIntegrationTestSuite))
}
