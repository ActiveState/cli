package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/suite"
)

type PrepareIntegrationTestSuite struct {
	suite.Suite
}

func (suite *PrepareIntegrationTestSuite) TestPrepare() {
	if !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestPrepare when not running on CI or on MacOS, as it modifies PATH")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("_prepare"),
		// e2e.AppendEnv(fmt.Sprintf("ACTIVESTATE_CLI_CONFIGDIR=%s", ts.Dirs.Work)),
	)
	cp.ExpectExitCode(0)
	suite.AssertConfig(filepath.Join(ts.Dirs.Cache, "bin"))
}

func (suite *PrepareIntegrationTestSuite) AssertConfig(target string) {
	if runtime.GOOS != "windows" {
		// Test bashrc
		homeDir, err := os.UserHomeDir()
		suite.Require().NoError(err)

		bashContents := fileutils.ReadFileUnsafe(filepath.Join(homeDir, ".bashrc"))
		suite.Contains(string(bashContents), constants.RCAppendDefaultStartLine, "bashrc should contain our RC Append Start line")
		suite.Contains(string(bashContents), constants.RCAppendDefaultStopLine, "bashrc should contain our RC Append Stop line")
		suite.Contains(string(bashContents), target, "bashrc should contain our target dir")
	} else {
		// Test registry
		out, err := exec.Command("reg", "query", `HKCU\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)
		path, err := osutil.GetLongPathName(target)
		suite.Require().NoError(err)
		suite.Contains(string(out), path, "Windows system PATH should contain our target dir")
	}
}

func TestPrepareIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PrepareIntegrationTestSuite))
}
