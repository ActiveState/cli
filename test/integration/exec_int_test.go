package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ExecIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ExecIntegrationTestSuite) createProjectFile(ts *e2e.Session) {
	ts.PrepareActiveStateYAML(strings.TrimSpace(`
		project: https://platform.activestate.com/ActiveState-CLI/Python3?commitID=fbc613d6-b0b1-4f84-b26e-4aa5869c4e54
	`))
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

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("exec", testScript),
	)
	cp.ExpectExitCode(0)
	output := cp.Snapshot()
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

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("exec", "--", testScript),
	)
	cp.ExpectExitCode(42)
}

func (suite *ExecIntegrationTestSuite) TestExec_Args() {
	suite.OnlyRunForTags(tagsuite.Exec)
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
	err := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(err)

	err = os.Chmod(testScript, 0777)
	suite.Require().NoError(err)

	args := []string{
		"firstArgument",
		"secondArgument",
		"thirdArgument",
	}

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("exec", "--", fmt.Sprintf("%s", testScript), args[0], args[1], args[2]),
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

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("exec", "--", fmt.Sprintf("%s", testScript)),
	)
	cp.SendLine("ActiveState")
	cp.Expect("Hello ActiveState!")
	cp.ExpectExitCode(0)
}

func TestExecIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExecIntegrationTestSuite))
}
