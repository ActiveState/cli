package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ShimIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ShimIntegrationTestSuite) createProjectFile(ts *e2e.Session) {
	ts.PrepareActiveStateYAML(strings.TrimSpace(`
		project: https://platform.activestate.com/ActiveState-CLI/Python3?commitID=fbc613d6-b0b1-4f84-b26e-4aa5869c4e54
	`))
}

func (suite *ShimIntegrationTestSuite) TestShim_Environment() {
	suite.OnlyRunForTags(tagsuite.Shim)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

	scriptBlock := `echo $PATH`
	filename := fmt.Sprintf("%s/%s.sh", ts.Dirs.Work, suite.T().Name())
	if runtime.GOOS == "windows" {
		scriptBlock = `echo %PATH%`
		filename = fmt.Sprintf("%s/%s.bat", ts.Dirs.Work, suite.T().Name())
	}

	testScript := filepath.Join(filename)
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail)

	err := os.Chmod(testScript, 0777)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", testScript),
	)
	cp.ExpectExitCode(0)
	output := cp.TrimmedSnapshot()
	if !strings.Contains(output, ts.Dirs.Bin) {
		suite.T().Fatal("PATH was not updated to contain cache directory")
	}
}

func (suite *ShimIntegrationTestSuite) TestShim_ExitCode() {
	suite.OnlyRunForTags(tagsuite.Shim)
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
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail)

	err := os.Chmod(testScript, 0777)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", "--", testScript),
	)
	cp.ExpectExitCode(42)
}

func (suite *ShimIntegrationTestSuite) TestShim_Args() {
	suite.OnlyRunForTags(tagsuite.Shim)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

	scriptBlock := `
for i; do
    echo $i
done
echo "Number of arguments: $#"
`

	filename := fmt.Sprintf("%s/%s.sh", ts.Dirs.Work, suite.T().Name())
	if runtime.GOOS == "windows" {
		scriptBlock = `
		set argCount=0
		for %%a in (%*) do (
			echo %%a
			set /A argCount+=1
		)
		echo Number of arguments: %argCount%`
		filename = fmt.Sprintf("%s/%s.bat", ts.Dirs.Work, suite.T().Name())
	}

	testScript := filepath.Join(filename)
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail)

	err := os.Chmod(testScript, 0777)
	suite.Require().NoError(err)

	args := []string{
		"firstArgument",
		"secondArgument",
		"thirdArgument",
	}

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", "--", fmt.Sprintf("%s", testScript), args[0], args[1], args[2]),
	)
	cp.Expect(args[0])
	cp.Expect(args[1])
	cp.Expect(args[2])
	cp.Expect(fmt.Sprintf("Number of arguments: %d", len(args)))
	cp.ExpectExitCode(0)
}

func (suite *ShimIntegrationTestSuite) TestShim_Input() {
	suite.OnlyRunForTags(tagsuite.Shim)
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
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail)

	err := os.Chmod(testScript, 0777)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", "--", fmt.Sprintf("%s", testScript)),
	)
	cp.SendLine("ActiveState")
	cp.Expect("Hello ActiveState!")
	cp.ExpectExitCode(0)
}

func (suite *ShimIntegrationTestSuite) TestShim_SystemPython() {
	suite.OnlyRunForTags(tagsuite.Shim)
	_, err := exec.LookPath("python3")
	if err != nil {
		suite.T().Skip("Cannot run test if system does not have python installation")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	scriptBlock := `print("Hello World!")`

	testScript := filepath.Join(fmt.Sprintf("%s/%s.py", ts.Dirs.Work, suite.T().Name()))
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail)

	cp := ts.Spawn("shim", "--", "python3", testScript)
	cp.Expect("Hello World!")
	cp.ExpectExitCode(0)
}

func (suite *ShimIntegrationTestSuite) TestShim_NoDoubleDash() {
	suite.OnlyRunForTags(tagsuite.Shim)
	_, err := exec.LookPath("python3")
	if err != nil {
		suite.T().Skip("Cannot run test if system does not have python installation")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	scriptBlock := `print("Hello World!")`

	testScript := filepath.Join(fmt.Sprintf("%s/%s.py", ts.Dirs.Work, suite.T().Name()))
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail)

	cp := ts.Spawn("shim", "python3", testScript)
	cp.Expect("Hello World!")
	cp.ExpectExitCode(0)
}

func TestShimIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShimIntegrationTestSuite))
}
