package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ExecIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ExecIntegrationTestSuite) createProjectFile(ts *e2e.Session) {
	ts.PrepareProject("ActiveState-CLI/Python3", "fbc613d6-b0b1-4f84-b26e-4aa5869c4e54")
}

func (suite *ExecIntegrationTestSuite) TestExec_Environment() {
	suite.OnlyRunForTags(tagsuite.Exec)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

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

	suite.createProjectFile(ts)

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

func (suite *ExecIntegrationTestSuite) TestExec_Args() {
	suite.OnlyRunForTags(tagsuite.Exec)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

	args := []string{
		"firstArgument",
		"secondArgument",
		"thirdArgument",
	}

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("exec", "--", "python3", "-c",
			"import sys; print(sys.argv); print(\"Number of arguments: %d\" % (len(sys.argv) - 1))",
			args[0], args[1], args[2]),
	)
	cp.Expect(args[0])
	cp.Expect(args[1])
	cp.Expect(args[2])
	cp.Expect(fmt.Sprintf("Number of arguments: %d", len(args)))
	cp.ExpectExitCode(0)
}

func (suite *ExecIntegrationTestSuite) TestExec_Input() {
	suite.OnlyRunForTags(tagsuite.Exec)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

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

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Python-3.9", pythonDir))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "--path", pythonDir, "--", "bash", "-c", "which python3"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Operating on project ActiveState-CLI/Python-3.9", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectRe(regexp.MustCompile("cache/[0-9A-Fa-f]+/usr/bin/python3").String())
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "echo", "python3", "--path", pythonDir, "--", "--path", "doesNotExist", "--", "extra"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("python3 --path doesNotExist -- extra")
	cp.ExpectExitCode(0)

}

func (suite *ExecIntegrationTestSuite) TestExecPerlArgs() {
	suite.OnlyRunForTags(tagsuite.Exec)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Perl-5.32", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	suite.NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "testargs.pl"), []byte(`
printf "Argument: '%s'.\n", $ARGV[0];
`)))

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "perl", "testargs.pl", "<3"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Argument: '<3'", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
}

func TestExecIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExecIntegrationTestSuite))
}
