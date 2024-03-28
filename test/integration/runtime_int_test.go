package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"
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

	if value, set := os.LookupEnv(constants.DisableRuntime); set {
		os.Setenv(constants.DisableRuntime, "false")
		defer os.Setenv(constants.DisableRuntime, value)
	}

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
	suite.OnlyRunForTags(tagsuite.Interrupt)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/test-interrupt-small-python#863c45e2-3626-49b6-893c-c15e85a17241", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	targetDir := target.ProjectDirToTargetDir(ts.Dirs.Work, ts.Dirs.Cache)
	pythonExe := filepath.Join(setup.ExecDir(targetDir), "python3"+osutils.ExeExtension)
	cp = ts.SpawnCmd(pythonExe, "-c", `print(__import__('sys').version)`)
	cp.Expect("3.8.8")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("pull"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false",
			constants.RuntimeSetupWaitEnvVarName+"=true"),
	)
	time.Sleep(30 * time.Second)
	cp.SendCtrlC() // cancel pull/update

	cp = ts.SpawnCmd(pythonExe, "-c", `print(__import__('sys').version)`)
	cp.Expect("3.8.8") // current runtime still works
	cp.ExpectExitCode(0)
}

func (suite *RuntimeIntegrationTestSuite) TestInUse() {
	suite.OnlyRunForTags(tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Perl-5.32", ".")
	cp.Expect("Skipping runtime setup")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("shell"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("perl")
	time.Sleep(1 * time.Second) // allow time for perl to start up

	cp2 := ts.SpawnWithOpts(
		e2e.OptArgs("install", "DateTime"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp2.Expect("currently in use", termtest.OptExpectTimeout(15*time.Second))
	cp2.Expect("perl")
	cp2.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()

	cp.SendCtrlC()
	cp.SendLine("exit")
	cp.ExpectExit() // code can vary depending on shell; just assert process finished
}

func TestRuntimeIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RuntimeIntegrationTestSuite))
}
