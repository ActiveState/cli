package integration

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/artifacts"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ArtifactsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ArtifactsIntegrationTestSuite) TestArtifacts() {
	suite.OnlyRunForTags(tagsuite.Artifacts)
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
		cp = ts.Spawn("artifacts")
		cp.Expect("Operating on project ActiveState-CLI/Python-With-Custom-Builds, located at")
		cp.Expect("CentOS")
		cp.Expect("Docker Image")
		cp.Expect("Installer")
		cp.Expect("macOS")
		cp.Expect("No artifacts")
		cp.Expect("Windows")
		cp.Expect("Signed Installer")
		cp.Expect(".exe")
		cp.Expect("To download artifacts run")
		cp.ExpectExitCode(0)
	})

	suite.Run("--all flag", func() {
		cp = ts.Spawn("artifacts", "--all")
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
		cp.Expect("To download artifacts run")
		cp.ExpectExitCode(0)
	})

	suite.Run("json without flags", func() {
		cp = ts.SpawnWithOpts(e2e.OptArgs("artifacts", "--output=json"), e2e.OptTermTest(termtest.OptRows(100)))
		cp.ExpectExitCode(0)

		output := artifacts.StructuredOutput{}
		out := strings.TrimLeft(cp.StrippedSnapshot(), locale.T("notice_runtime_disabled"))
		suite.Require().NoError(json.Unmarshal([]byte(out), &output), ts.DebugMessage(""))

		suite.Equal(3, len(output.Platforms))
		for _, platform := range output.Platforms {
			if !strings.HasPrefix(platform.Name, "macOS") {
				suite.Greater(len(platform.Artifacts), 0)
			}
			suite.Equal(0, len(platform.Packages))
		}
	})

	suite.Run("json with --all flag", func() {
		cp = ts.SpawnWithOpts(e2e.OptArgs("artifacts", "--output=json", "--all"), e2e.OptTermTest(termtest.OptRows(100)))
		cp.ExpectExitCode(0)

		output := artifacts.StructuredOutput{}
		out := strings.TrimLeft(cp.StrippedSnapshot(), locale.T("notice_runtime_disabled"))
		suite.Require().NoError(json.Unmarshal([]byte(out), &output), ts.DebugMessage(""))

		suite.Equal(3, len(output.Platforms))
		for _, platform := range output.Platforms {
			if !strings.HasPrefix(platform.Name, "macOS") {
				suite.Greater(len(platform.Artifacts), 0)
			}
			suite.Greater(len(platform.Packages), 0)
		}
	})
}

func (suite *ArtifactsIntegrationTestSuite) TestArtifacts_Remote() {
	suite.OnlyRunForTags(tagsuite.Artifacts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.Run("Namespace only", func() {
		cp := ts.Spawn("artifacts", "--namespace", "ActiveState-CLI/Python-With-Custom-Builds")
		cp.Expect("CentOS")
		cp.Expect("Docker Image")
		cp.Expect("Installer")
		cp.Expect("macOS")
		cp.Expect("No artifacts")
		cp.Expect("Windows")
		cp.Expect("Signed Installer")
		cp.Expect(".exe")
		cp.Expect("To download artifacts run")
		suite.Assert().NotContains(cp.Snapshot(), "Operating on project")
		cp.ExpectExitCode(0)
	})

	suite.Run("Namespace and commit ID", func() {
		cp := ts.Spawn("artifacts", "--namespace", "ActiveState-CLI/Python-With-Custom-Builds", "--commit", "993454c7-6613-4b1a-8981-1cee43cc249e")
		cp.Expect("CentOS")
		cp.Expect("Docker Image")
		cp.Expect("Installer")
		cp.Expect("macOS")
		cp.Expect("No artifacts")
		cp.Expect("Windows")
		cp.Expect("Signed Installer")
		cp.Expect(".exe")
		cp.Expect("To download artifacts run")
		cp.ExpectExitCode(0)
	})
}

type Platforms struct {
	Platforms []Platform `json:"platforms"`
}
type Platform struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Packages Packages `json:"packages"`
}

type Packages []Build

type Build struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (suite *ArtifactsIntegrationTestSuite) TestArtifacts_Download() {
	suite.OnlyRunForTags(tagsuite.Artifacts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Python-With-Custom-Builds", "993454c7-6613-4b1a-8981-1cee43cc249e")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	var buildID string
	if runtime.GOOS == "windows" {
		// On Windows we need the specific build ID as the terminal buffer is not
		// large enough to display all the builds
		buildID = "dbf05bf8-4b2e-5560-a329-b5b70bc7b0fa"
	} else {
		buildID = suite.extractBuildID(ts, "bzip2@1.0.8", "")
	}
	suite.Require().NotEmpty(buildID)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("artifacts", "dl", buildID, "."),
	)
	cp.Expect("Operating on project ActiveState-CLI/Python-With-Custom-Builds, located at")
	cp.Expect("Downloaded bzip2", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
	require.FileExists(suite.T(), filepath.Join(ts.Dirs.Work, "bzip2-1.0.8.tar.gz"))
}

func (suite *ArtifactsIntegrationTestSuite) TestArtifacts_Download_Remote() {
	suite.OnlyRunForTags(tagsuite.Artifacts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	var buildID string
	if runtime.GOOS == "windows" {
		// On Windows we need the specific build ID as the terminal buffer is not
		// large enough to display all the builds
		buildID = "dbf05bf8-4b2e-5560-a329-b5b70bc7b0fa"
	} else {
		buildID = suite.extractBuildID(ts, "bzip2@1.0.8", "ActiveState-CLI/Python-With-Custom-Builds")
	}
	suite.Require().NotEmpty(buildID)

	cp := ts.Spawn("artifacts", "dl", buildID, ".", "--namespace", "ActiveState-CLI/Python-With-Custom-Builds")
	cp.Expect("Downloaded bzip2", e2e.RuntimeSourcingTimeoutOpt)
	suite.Assert().NotContains(cp.Snapshot(), "Operating on project")
	cp.ExpectExitCode(0)
	require.FileExists(suite.T(), filepath.Join(ts.Dirs.Work, "bzip2-1.0.8.tar.gz"))
}

func (suite *ArtifactsIntegrationTestSuite) extractBuildID(ts *e2e.Session, name string, namespace string) string {
	args := []string{"builds", "--all", "--output=json"}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}

	cp := ts.SpawnWithOpts(
		e2e.OptArgs(args...),
	)
	cp.Expect(`"}`)
	cp.ExpectExitCode(0)

	var platforms Platforms
	suite.Require().NoError(json.Unmarshal([]byte(cp.Output()), &platforms))

	var platformID string
	switch runtime.GOOS {
	case "windows":
		platformID = constants.Win10Bit64UUID
	case "darwin":
		platformID = constants.MacBit64UUID
	case "linux":
		platformID = constants.LinuxBit64UUID
	}

	for _, platform := range platforms.Platforms {
		if platform.ID != platformID {
			continue
		}

		for _, build := range platform.Packages {
			if build.Name == name {
				return build.ID
			}
		}
	}

	return ""
}

func TestArtifactsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ArtifactsIntegrationTestSuite))
}
