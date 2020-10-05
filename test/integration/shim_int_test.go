package integration

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type ShimIntegrationTestSuite struct {
	suite.Suite
}

func (suite *ShimIntegrationTestSuite) createProjectFile(ts *e2e.Session) {
	ts.PrepareActiveStateYAML(strings.TrimSpace(`
		project: https://platform.activestate.com/ActiveState-CLI/Python3?commitID=fbc613d6-b0b1-4f84-b26e-4aa5869c4e54
	`))
}

func (suite *ShimIntegrationTestSuite) TestShim_Basic() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

	scriptBlock := `
print("Hello World!")
	`

	testScript := filepath.Join(fmt.Sprintf("%s/%s.py", ts.Dirs.Work, suite.T().Name()))
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail.ToError())

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", "--", "python3", fmt.Sprintf("%s", testScript)),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Hello World!")
	cp.ExpectExitCode(0)
}

func (suite *ShimIntegrationTestSuite) TestShim_ExitCode() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

	scriptBlock := `
import sys
sys.exit(42)
`

	testScript := filepath.Join(fmt.Sprintf("%s/%s.py", ts.Dirs.Work, suite.T().Name()))
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail.ToError())

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", "--", "python3", fmt.Sprintf("%s", testScript)),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectExitCode(42)
}

func (suite *ShimIntegrationTestSuite) TestShim_Args() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

	scriptBlock := `
import sys
# Remove script path from argument list
sys.argv.pop(0)

# Printing str(sys.argv) introduces formatting that the 
# integration tests do not like
arg_1 = sys.argv[0]
arg_2 = sys.argv[1]
arg_3 = sys.argv[2]

print("Number of arguments:", len(sys.argv))
print("Your arguments are: {}, {}, {}".format(arg_1, arg_2, arg_3))
`

	testScript := filepath.Join(fmt.Sprintf("%s/%s.py", ts.Dirs.Work, suite.T().Name()))
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail.ToError())

	args := []string{
		"firstArgument",
		"secondArgument",
		"thirdArgument",
	}
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", "--", "python3", fmt.Sprintf("%s", testScript), args[0], args[1], args[2]),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Number of arguments: 3")
	cp.ExpectLongString(fmt.Sprintf("Your arguments are: %s, %s, %s", args[0], args[1], args[2]))
	cp.ExpectExitCode(0)
}

func (suite *ShimIntegrationTestSuite) TestShim_Input() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

	scriptBlock := `
name = input("Enter your name: ")
print("Hello {}!".format(name))
`

	testScript := filepath.Join(fmt.Sprintf("%s/%s.py", ts.Dirs.Work, suite.T().Name()))
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail.ToError())

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", "--", "python3", fmt.Sprintf("%s", testScript)),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.SendLine("ActiveState")
	cp.Expect("Hello ActiveState!")
	cp.ExpectExitCode(0)
}

func (suite *ShimIntegrationTestSuite) TestShim_SystemPython() {
	_, err := exec.LookPath("python3")
	if err != nil {
		suite.T().Skip("Cannot run test if system does not have python installation")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	scriptBlock := `
print("Hello World!")
`

	testScript := filepath.Join(fmt.Sprintf("%s/%s.py", ts.Dirs.Work, suite.T().Name()))
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail.ToError())

	cp := ts.Spawn("shim", "--", "python3", testScript)
	cp.Expect("Hello World!")
	cp.ExpectExitCode(0)
}

func (suite *ShimIntegrationTestSuite) TestShim_NoDoubleDash() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts)

	scriptBlock := `
print("Hello World!")
	`

	testScript := filepath.Join(fmt.Sprintf("%s/%s.py", ts.Dirs.Work, suite.T().Name()))
	fail := fileutils.WriteFile(testScript, []byte(scriptBlock))
	suite.Require().NoError(fail.ToError())

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("shim", "python3", fmt.Sprintf("%s", testScript)),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Hello World!")
	cp.ExpectExitCode(0)
}

func TestShimIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShimIntegrationTestSuite))
}
