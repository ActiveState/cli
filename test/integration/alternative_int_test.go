package integration

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type AlternativeArtifactIntegrationTestSuite struct {
	suite.Suite
}

func (suite *AlternativeArtifactIntegrationTestSuite) TestActivateRuby() {
	suite.T().Skip("requires a working PR branch for now.")
	if runtime.GOOS != "linux" {
		suite.T().Skip("only works on linux")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	extraEnv := e2e.AppendEnv(
		"ACTIVESTATE_API_HOST=pr3134.activestate.build",
		"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
	)

	// Download artifacts but interrupt installation step
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "install", "martind-stage/ruby"),
		extraEnv,
	)

	// TODO interrupt a download, and ensure that download is retried!
	cp.Expect("Downloading")
	cp.Expect("6 / 6")
	cp.Expect("Installing")
	cp.SendCtrlC()
	cp.ExpectNotExitCode(0)

	// On activation, nothing is downloaded, but installation is completed
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate", "martind-stage/ruby", "--path", ts.Dirs.Work),
		extraEnv,
	)
	cp.Expect("Installing")
	cp.Expect("6 / 6")
	cp.Expect("activated state")

	cp.SendLine(`ruby -e 'puts "      world\rhello"'`)
	cp.Expect("hello world")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// Completely cached activation: no file needs to be downloaded or installed
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate", "martind-stage/ruby", "--path", ts.Dirs.Work),
		extraEnv,
	)
	cp.Expect("activated state")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
	suite.NotContains(cp.TrimmedSnapshot(), "Downloading required artifacts")
	suite.NotContains(cp.TrimmedSnapshot(), "Installing")

	// Only one cached download missing
	cachedArtifacts, err := ioutil.ReadDir(filepath.Join(ts.Dirs.Cache, "artifacts"))
	suite.Require().NoError(err, "listing cached artifacts")
	suite.Len(cachedArtifacts, 6, "expected six cached artifacts")

	err = os.RemoveAll(filepath.Join(ts.Dirs.Cache, "artifacts", cachedArtifacts[0].Name()))
	suite.Require().NoError(err, "removing a single artifact")
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate", "martind-stage/ruby", "--path", ts.Dirs.Work),
		extraEnv,
	)
	cp.Expect("Downloading")
	cp.Expect("1 / 1")
	cp.Expect("Installing")
	cp.Expect("6 / 6")
	cp.Expect("activated state")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func TestAlternativeArtifactIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AlternativeArtifactIntegrationTestSuite))
}
