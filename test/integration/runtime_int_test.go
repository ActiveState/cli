package integration

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/project"
	rt "github.com/ActiveState/cli/pkg/runtime"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
)

// Disabled due to DX-1514
/*func TestOfflineInstaller(t *testing.T) {
	// Each artifact of the form UUID.tar.gz has the following structure:
	// - runtime.json (empty)
	// - tmp (directory)
	//   - [number] (file)
	// The numbered file is the key to the following maps.
	testArtifacts := map[string]strfmt.UUID{
		"1":  "74D554B3-6B0F-434B-AFE2-9F2F0B5F32BA",
		"2":  "87ADD1B0-169D-4C01-8179-191BB9910799",
		"3":  "5D8D933F-09FA-45A3-81FF-E6F33E91C9ED",
		"4":  "992B8488-C61D-433C-ADF2-D76EBD8DAE59",
		"5":  "2C36A315-59ED-471B-8629-2663ECC95476",
		"6":  "57E8EAF4-F7EE-4BEF-B437-D9F0A967BA52",
		"7":  "E299F10C-7B5D-4B25-B821-90E30193A916",
		"8":  "F95C0ECE-9F69-4998-B83F-CE530BACD468",
		"9":  "CAC9708D-FAA6-4295-B640-B8AA41A8AABC",
		"10": "009D20C9-0E38-44E8-A095-7B6FEF01D7DA",
	}
	const artifactsPerArtifact = 2 // files/artifacts per artifact.tar.gz

	dir, err := os.MkdirTemp("", "")
	suite.Require().NoError(err)
	defer os.RemoveAll(dir)

	artifactsDir := filepath.Join(osutil.GetTestDataDir(), "offline-runtime")
	offlineTarget := target.NewOfflineTarget(nil, dir, artifactsDir)

	analytics := blackhole.New()
	mockProgress := &testhelper.MockProgressOutput{}
	logfile, err := buildlogfile.New(outputhelper.NewCatcher())
	suite.Require().NoError(err)
	eventHandler := events.NewRuntimeEventHandler(mockProgress, nil, logfile)

	rt, err := runtime.New(offlineTarget, analytics, nil, nil)
	suite.Require().Error(err)
	err = rt.Update(eventHandler)
	suite.Require().NoError(err)

	suite.Assert().False(mockProgress.BuildStartedCalled)
	suite.Assert().False(mockProgress.BuildCompletedCalled)
	suite.Assert().Equal(int64(0), mockProgress.BuildTotal)
	suite.Assert().Equal(0, mockProgress.BuildCurrent)
	suite.Assert().Equal(true, mockProgress.InstallationStartedCalled)
	suite.Assert().Equal(true, mockProgress.InstallationCompletedCalled)
	suite.Assert().Equal(int64(len(testArtifacts)), mockProgress.InstallationTotal)
	suite.Assert().Equal(len(testArtifacts)*artifactsPerArtifact, mockProgress.ArtifactStartedCalled)
	suite.Assert().Equal(2*len(testArtifacts)*artifactsPerArtifact, mockProgress.ArtifactIncrementCalled) // start and stop each have one count
	suite.Assert().Equal(len(testArtifacts)*artifactsPerArtifact, mockProgress.ArtifactCompletedCalled)
	suite.Assert().Equal(0, mockProgress.ArtifactFailureCalled)

	for filename := range testArtifacts {
		filename := filepath.Join(dir, "tmp", filename) // each file is in a "tmp" dir in the archive
		suite.Assert().True(fileutils.FileExists(filename), "file '%s' was not extracted from its artifact", filename)
	}
}*/

type RuntimeIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *RuntimeIntegrationTestSuite) TestInterruptSetup() {
	if runtime.GOOS == "windows" {
		// https://activestatef.atlassian.net/browse/DX-2926
		suite.T().Skip("interrupting on windows is currently broken when ran via CI")
	}

	suite.OnlyRunForTags(tagsuite.Interrupt)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/test-interrupt-small-python#863c45e2-3626-49b6-893c-c15e85a17241", ".")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	proj, err := project.FromPath(ts.Dirs.Work)
	suite.Require().NoError(err)

	execPath := rt.ExecutorsPath(filepath.Join(ts.Dirs.Cache, runtime_helpers.DirNameFromProjectDir(proj.Dir())))
	pythonExe := filepath.Join(execPath, "python3"+osutils.ExeExtension)

	cp = ts.SpawnCmd(pythonExe, "-c", `print(__import__('sys').version)`)
	cp.Expect("3.8.8")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("pull")
	cp.Expect("Downloading")
	cp.SendCtrlC() // cancel pull/update
	cp.ExpectExitCode(1)

	cp = ts.SpawnCmd(pythonExe, "-c", `print(__import__('sys').version)`)
	cp.Expect("3.8.8") // current runtime still works
	cp.ExpectExitCode(0)
	ts.IgnoreLogErrors() // Should see an error related to the interrupted setup
}

func (suite *RuntimeIntegrationTestSuite) TestInUse() {
	if runtime.GOOS == "windows" {
		// https://activestatef.atlassian.net/browse/DX-2926
		suite.T().Skip("interrupting on windows is currently broken when ran via CI")
	}
	if runtime.GOOS == "darwin" {
		return // gopsutil errors on later versions of macOS (DX-2723)
	}
	suite.OnlyRunForTags(tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Empty", "b55d0e63-db48-43c4-8341-e2b7a1cc134c")

	cp := ts.Spawn("shell")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("perl")
	time.Sleep(1 * time.Second) // allow time for perl to start up

	cp2 := ts.Spawn("install", "DateTime")
	cp2.Expect("the runtime for this project is in use", e2e.RuntimeSourcingTimeoutOpt)
	cp2.Expect("perl")
	cp2.ExpectExitCode(0)

	cp.SendCtrlC()
	cp.SendLine("exit")
	cp.ExpectExit() // code can vary depending on shell; just assert process finished
}

func (suite *RuntimeIntegrationTestSuite) TestBuildInProgress() {
	suite.T().Skip("Publishing is taking a backseat to buildscripts and dynamic imports")
	suite.OnlyRunForTags(tagsuite.BuildInProgress)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	// Publish a new ingredient revision, which, when coupled with `state install --ts now`, will
	// force a build.
	// The ingredient is a tarball comprising:
	//   1. An empty, executable "configure" script (emulating autotools).
	//   2. A simple Makefile with "all", "check", and "install" rules.
	//   3. A simple "main.c" file, whose compiled executable prints "Hello world!".
	cp := ts.Spawn("publish", "--non-interactive",
		"--namespace", "private/"+e2e.PersistentUsername,
		"--name", "hello-world",
		"--version", "1.0.0",
		"--depend", "builder/autotools-builder@>=0", // for ./configure, make, make install
		"--depend", "internal/mingw-build-selector@>=0", // for Windows to use mingw's GCC
		filepath.Join(osutil.GetTestDataDir(), "hello-world-1.0.0.tar.gz"),
		"--edit") // publish a new revision each time, forcing a build
	cp.Expect("Successfully published")
	cp.ExpectExitCode(0)

	ts.PrepareEmptyProject()

	cp = ts.Spawn("install", "private/"+e2e.PersistentUsername+"/hello-world", "--ts", "now")
	cp.Expect("Build Log:")
	cp.Expect("Detailed Progress:")
	cp.Expect("Building")
	cp.Expect("All dependencies have been installed and verified", e2e.RuntimeBuildSourcingTimeoutOpt)
	cp.Expect("Added: private/" + e2e.PersistentUsername + "/hello-world")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("exec", "main")
	cp.Expect("Hello world!")
	cp.ExpectExitCode(0)
}

func (suite *RuntimeIntegrationTestSuite) TestIgnoreEnvironmentVars() {
	suite.OnlyRunForTags(tagsuite.Environment)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/small-python", ".")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	pythonPath := "my/path"

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "python3", "--", "-c", `print(__import__("os").environ["PYTHONPATH"])`),
		e2e.OptAppendEnv("PYTHONPATH="+pythonPath),
	)
	cp.ExpectExitCode(0)
	suite.Assert().NotContains(cp.Snapshot(), pythonPath)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "python3", "--", "-c", `print(__import__("os").environ["PYTHONPATH"])`),
		e2e.OptAppendEnv(
			"PYTHONPATH="+pythonPath,
			constants.IgnoreEnvEnvVarName+"=PYTHONPATH",
		))
	cp.Expect(pythonPath)
	cp.ExpectExitCode(0)
}

func (suite *RuntimeIntegrationTestSuite) TestRuntimeCache() {
	suite.OnlyRunForTags(tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("install", "shared/zlib")
	cp.Expect("Downloading")
	cp.ExpectExitCode(0)

	depot := filepath.Join(ts.Dirs.Cache, "depot")
	artifacts, err := fileutils.ListDirSimple(depot, true)
	suite.Require().NoError(err)

	cp = ts.Spawn("switch", "mingw") // should not remove cached shared/zlib artifact
	cp.ExpectExitCode(0)

	artifacts2, err := fileutils.ListDirSimple(depot, true)
	suite.Require().NoError(err)
	suite.Assert().Equal(artifacts, artifacts2, "shared/zlib should have remained in the cache")

	cp = ts.Spawn("config", "set", constants.RuntimeCacheSizeConfigKey, "0")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("switch", "main") // should remove cached shared/zlib artifact
	cp.ExpectExitCode(0)

	artifacts3, err := fileutils.ListDirSimple(depot, true)
	suite.Require().NoError(err)
	suite.Assert().NotEqual(artifacts, artifacts3, "shared/zlib should have been removed from the cache")
}

func TestRuntimeIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RuntimeIntegrationTestSuite))
}
