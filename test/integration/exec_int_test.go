package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
)

type ExecIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ExecIntegrationTestSuite) TestExec_Environment() {
	suite.OnlyRunForTags(tagsuite.Exec)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	scriptBlock := `echo ${PATH:0:500}`
	filename := fmt.Sprintf("%s/%s.sh", ts.Dirs.Work, suite.T().Name())
	if runtime.GOOS == "windows" {
		scriptBlock = `echo %PATH:~0,500%`
		filename = fmt.Sprintf("%s/%s.bat", ts.Dirs.Work, suite.T().Name())
	}

	testScript := filepath.Join(filename)
	err := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(err)

	err = os.Chmod(testScript, 0777)
	suite.Require().NoError(err)

	args := []string{"exec", "--", "bash", "-c", testScript}
	if runtime.GOOS == "windows" {
		args = []string{"exec", "--", "cmd", "/c", testScript}
	}
	cp := ts.SpawnWithOpts(
		e2e.OptArgs(args...),
	)
	cp.ExpectExitCode(0)
	output := cp.Output()
	suite.Contains(output, ts.Dirs.Bin, "PATH was not updated to contain cache directory, original PATH:", os.Getenv("PATH"))
}

func (suite *ExecIntegrationTestSuite) TestExec_ExitCode() {
	suite.OnlyRunForTags(tagsuite.Exec)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	scriptBlock := `exit 42`
	filename := fmt.Sprintf("%s/%s.sh", ts.Dirs.Work, suite.T().Name())
	if runtime.GOOS == "windows" {
		scriptBlock = `EXIT 42`
		filename = fmt.Sprintf("%s/%s.bat", ts.Dirs.Work, suite.T().Name())
	}

	testScript := filepath.Join(filename)
	err := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(err)

	err = os.Chmod(testScript, 0777)
	suite.Require().NoError(err)

	args := []string{"exec", "--", "bash", "-c", testScript}
	if runtime.GOOS == "windows" {
		args = []string{"exec", "--", "cmd", "/c", testScript}
	}
	cp := ts.SpawnWithOpts(
		e2e.OptArgs(args...),
	)
	cp.ExpectExitCode(42)
}

func (suite *ExecIntegrationTestSuite) TestExec_Input() {
	suite.OnlyRunForTags(tagsuite.Exec)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	scriptBlock := `
echo "Enter your name: "
read name
echo "Hello $name!"
`

	filename := fmt.Sprintf("%s/%s.sh", ts.Dirs.Work, suite.T().Name())
	if runtime.GOOS == "windows" {
		scriptBlock = `set /P name="Enter your name: "
		echo Hello %name%!`
		filename = fmt.Sprintf("%s/%s.bat", ts.Dirs.Work, suite.T().Name())
	}

	testScript := filepath.Join(filename)
	err := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(err)

	err = os.Chmod(testScript, 0777)
	suite.Require().NoError(err)

	args := []string{"exec", "--", "bash", "-c", testScript}
	if runtime.GOOS == "windows" {
		args = []string{"exec", "--", "cmd", "/c", testScript}
	}
	cp := ts.SpawnWithOpts(
		e2e.OptArgs(args...),
	)
	cp.SendLine("ActiveState")
	cp.Expect("Hello ActiveState!")
	cp.ExpectExitCode(0)
}

func (suite *ExecIntegrationTestSuite) TestExecWithPath() {
	if runtime.GOOS == "windows" {
		suite.T().Skip("Windows does not have `which` command")
	}
	suite.OnlyRunForTags(tagsuite.Exec)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	pythonDir := filepath.Join(ts.Dirs.Work, "MyPython3")

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python-3.9", pythonDir)
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "--path", pythonDir, "--", "bash", "-c", "which python3"),
	)
	cp.Expect("Operating on project ActiveState-CLI/Python-3.9", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectRe(regexp.MustCompile("cache/[0-9A-Fa-f]+/usr/bin/python3").String())
	cp.ExpectExitCode(0)

	cp = ts.Spawn("exec", "echo", "python3", "--path", pythonDir, "--", "--path", "doesNotExist", "--", "extra")
	cp.Expect("python3 --path doesNotExist -- extra")
	cp.ExpectExitCode(0)

}

func (suite *ExecIntegrationTestSuite) TestExeBatArguments() {
	suite.OnlyRunForTags(tagsuite.Exec)

	if runtime.GOOS != "windows" {
		suite.T().Skip("This test is only for windows")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	root := environment.GetRootPathUnsafe()
	reportBat := filepath.Join(root, "test", "integration", "testdata", "batarguments", "report.bat")
	suite.Require().FileExists(reportBat)

	inputs := []string{"a<b", "b>a", "hello world", "&whoami", "imnot|apipe", "%NotAppData%", "^NotEscaped", "(NotAGroup)"}
	outputs := `"` + strings.Join(inputs, `" "`) + `"`
	cp = ts.SpawnWithOpts(e2e.OptArgs(append([]string{"exec", reportBat, "--"}, inputs...)...))
	cp.Expect(outputs, termtest.OptExpectTimeout(5*time.Second))
	cp.ExpectExitCode(0)
}

func TestExecIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExecIntegrationTestSuite))
}
