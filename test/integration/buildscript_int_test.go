package integration

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runners/artifacts"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/termtest"
)

type BuildScriptIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *BuildScriptIntegrationTestSuite) TestBuildScript_NeedsReset() {
	suite.OnlyRunForTags(tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(fmt.Sprintf("project: https://%s/%s?commitID=%s\nconfig_version: %d\n",
		constants.DefaultAPIHost, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8", projectfile.ConfigVersion))

	cp := ts.Spawn("config", "set", constants.OptinBuildscriptsConfig, "true")
	cp.ExpectExitCode(0)

	suite.Require().NoFileExists(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName))

	cp = ts.Spawn("refresh")
	cp.Expect("Your project is missing its buildscript file")
	cp.ExpectExitCode(1)

	cp = ts.Spawn("reset", "LOCAL")
	cp.ExpectExitCode(0)

	suite.Require().FileExists(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName), ts.DebugMessage(""))
}

func (suite *BuildScriptIntegrationTestSuite) TestBuildScript_IngredientFunc() {
	suite.OnlyRunForTags(tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	projectURL := fmt.Sprintf("https://%s/%s?commitID=%s", constants.DefaultAPIHost, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")
	ts.PrepareActiveStateYAML(fmt.Sprintf("project: %s\nconfig_version: %d\n", projectURL, projectfile.ConfigVersion))

	var platformID string
	switch runtime.GOOS {
	case "windows":
		platformID = constants.Win10Bit64UUID
	case "darwin":
		platformID = constants.MacBit64UUID
	default:
		platformID = constants.LinuxBit64UUID
	}

	denoter := "```"
	ts.PrepareBuildScript(fmt.Sprintf(`
%s
Project: %s
Time: "2024-10-30T21:31:33.000Z"
%s
wheel = make_wheel(
	at_time = TIME,
	src = tag(
		plan = ingredient(
			build_deps = [
				Req(name = "python-module-builder", namespace = "builder", version = Gte(value = "0")),
				Req(name = "python", namespace = "language", version = Gte(value = "3")),
				Req(name = "setuptools", namespace = "language/python", version = Gte(value = "43.0.0")),
				Req(name = "wheel", namespace = "language/python", version = Gte(value = "0"))
			],
			src = [
				"sample_ingredient/**"
			]
		),
		tag = "platform:%s"
	)
)

main = wheel
`, denoter, projectURL, denoter, platformID))

	// Prepare sample ingredient source files
	root := environment.GetRootPathUnsafe()
	sampleSource := filepath.Join(root, "test", "integration", "testdata", "sample_ingredient")
	sampleTarget := filepath.Join(ts.Dirs.Work, "sample_ingredient")
	suite.Require().NoError(fileutils.Mkdir(sampleTarget))
	suite.Require().NoError(fileutils.CopyFiles(sampleSource, sampleTarget))

	// Create a new commit, which will use the source files to create an ingredient if it doesn't already exist
	cp := ts.Spawn("commit")
	cp.ExpectExitCode(0, e2e.RuntimeSolvingTimeoutOpt, termtest.OptExpectErrorMessage(ts.DebugMessage("")))

	// Running commit again should say there are no changes
	// If this fails then there's likely an issue with calculating the file hash, or checking whether an ingredient
	// already exists with the given hash.
	cp = ts.Spawn("commit")
	cp.Expect("no new changes")
	cp.ExpectExitCode(0)

	// Commit should've given us the hash
	suite.Contains(string(fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName))), "hash_readonly")

	// Wait for build
	var out artifacts.StructuredOutput
	suite.Require().NoError(rtutils.Timeout(func() error {
		for {
			cp = ts.Spawn("artifacts", "--output=json")
			if err := cp.ExpectExitCode(0, termtest.OptExpectSilenceErrorHandler()); err != nil {
				return err
			}

			if err := json.Unmarshal(AssertValidJSON(suite.T(), cp), &out); err != nil {
				return err
			}
			if out.BuildComplete {
				break
			}
			time.Sleep(time.Second * 5)
		}
		return nil
	}, e2e.RuntimeBuildSourcingTimeout), ts.DebugMessage(""))

	// Ensure build didn't fail
	suite.False(out.HasFailedArtifacts)
	suite.Empty(out.Platforms[0].Artifacts[0].Errors, "Log: %s", out.Platforms[0].Artifacts[0].LogURL)

	// Download the wheel artifact that was produced from our source ingredient
	cp = ts.Spawn("artifacts", "dl", "--output=json", out.Platforms[0].Artifacts[0].ID)
	cp.ExpectExitCode(0)

	var path string
	suite.Require().NoError(json.Unmarshal(AssertValidJSON(suite.T(), cp), &path))

	// Read wheel archive and ensure it contains the expected files
	zipReader, err := zip.OpenReader(path)
	suite.Require().NoError(err)
	defer zipReader.Close()
	files := map[string]struct{}{}
	for _, f := range zipReader.File {
		files[f.Name] = struct{}{}
	}
	suite.Contains(files, "sample_activestate/__init__.py")
	suite.Contains(files, "sample_activestate-1.0.0.dist-info/WHEEL")
}

func TestBuildScriptIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BuildScriptIntegrationTestSuite))
}
