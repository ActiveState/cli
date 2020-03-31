package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

type ActivateIntegrationTestSuite struct {
	e2e.Suite
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3() {
	suite.activatePython("3")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3_zsh() {
	suite.activatePython("3", "SHELL=zsh")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython2() {
	suite.activatePython("2")
}

func (suite *ActivateIntegrationTestSuite) TestActivateWithoutRuntime() {
	suite.LoginAsPersistentUser()

	cp := suite.Spawn("activate", "ActiveState-CLI/Python3")
	defer cp.Close()
	cp.Expect("Where would you like to checkout")
	cp.SendLine(suite.WorkDirectory())
	cp.Expect("activated state", 20*time.Second)
	cp.WaitForInput(10 * time.Second)

	cp.SendLine("exit 123")
	cp.ExpectExitCode(123, 10*time.Second)
}

func (suite *ActivateIntegrationTestSuite) TestActivatePythonByHostOnly() {
	if runtime.GOOS != "linux" {
		suite.T().Skip("not currently testing this OS")
	}

	suite.LoginAsPersistentUser()

	projectName := "Python-LinuxWorks"
	cp := suite.Spawn("activate", "cli-integration-tests/"+projectName, "--path="+suite.WorkDirectory())
	defer cp.Close()

	cp.Expect("Activating state")
	cp.Expect("activated state", 120*time.Second)
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) activatePython(version string, extraEnv ...string) {
	if runtime.GOOS == "windows" {
		suite.T().Skip("suite.AppendEnv() does not work on windows currently.  Skipping this test.")
	}

	// temp skip // pythonExe := "python" + version

	suite.LoginAsPersistentUser()

	cp := suite.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/Python"+version),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		e2e.AppendEnv(extraEnv...),
	)
	defer cp.Close()
	cp.Expect("Where would you like to checkout")
	cp.SendLine(suite.Session.Dirs.Work)
	cp.Expect("Downloading", 20*time.Second)
	cp.Expect("Installing", 120*time.Second)
	cp.Expect("activated state", 120*time.Second)

	// ensure that terminal contains output "Installing x/y" with x, y numbers and x=y
	installingString := regexp.MustCompile(
		"Installing *([0-9]+) */ *([0-9]+)",
	).FindAllStringSubmatch(cp.TrimmedSnapshot(), 1)
	suite.Require().Len(installingString, 1, "no match for Installing x / x in\n%s", cp.TrimmedSnapshot())
	suite.Require().Equalf(
		installingString[0][1], installingString[0][2],
		"expected all artifacts are reported to be installed, got %s", installingString[0][0],
	)

	// ensure that shell is functional
	cp.WaitForInput()

	// test python
	// Temporarily skip these lines until MacOS on Python builds with correct copyright
	// temp skip // cp.SendLine(pythonExe + " -c \"import sys; print(sys.copyright)\"")
	// temp skip // cp.Expect("ActiveState Software Inc.")

	// temp skip // cp.SendLine(pythonExe + " -c \"import pytest; print(pytest.__doc__)\"")
	// temp skip // cp.Expect("unit and functional testing")

	// de-activate shell
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3_Forward() {
	var project string
	if runtime.GOOS == "darwin" {
		project = "Activate-MacOS"
	} else {
		project = "Python3"
	}

	contents := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/%s"
branch: %s
version: %s
`, project, constants.BranchName, constants.Version))

	suite.PrepareActiveStateYAML(contents)

	fmt.Printf("login \n")
	suite.LoginAsPersistentUser()
	fmt.Printf("logged in \n")

	// Ensure we have the most up to date version of the project before activating
	cp := suite.SpawnWithOpts(
		e2e.WithArgs("pull"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	defer cp.Close()
	cp.Expect("Your activestate.yaml has been updated to the latest version available")
	cp.Expect("If you have any active instances of this project open in other terminals")
	cp.ExpectExitCode(0)

	c2 := suite.Spawn("activate")
	defer c2.Close()
	c2.Expect(fmt.Sprintf("Activating state: ActiveState-CLI/%s", project))

	// not waiting for activation, as we test that part in a different test
	c2.WaitForInput()
	c2.SendLine("exit")
	c2.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) testOutput(method string) {
	suite.LoginAsPersistentUser()
	cp := suite.Spawn("activate", "ActiveState-CLI/Python3", "--output", method)
	defer cp.Close()
	cp.Expect("Where would you like to checkout")
	cp.SendLine(cp.WorkDirectory())
	cp.Expect("[activated-JSON]")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_Subdir() {
	fail := fileutils.Mkdir(suite.WorkDirectory(), "foo", "bar", "baz")
	suite.Require().NoError(fail.ToError())

	// Create the project file at the root of the temp dir
	content := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/Python3"
branch: %s
version: %s
`, constants.BranchName, constants.Version))

	suite.PrepareActiveStateYAML(content)

	// Pull to ensure we have an up to date config file
	cp := suite.Spawn("pull")
	defer cp.Close()
	cp.Expect("Your activestate.yaml has been updated to the latest version available")
	cp.Expect("If you have any active instances of this project open in other terminals")
	cp.ExpectExitCode(0)

	// Activate in the subdirectory
	c2 := suite.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.WithWorkDirectory(filepath.Join(suite.WorkDirectory(), "foo", "bar", "baz")),
	)
	defer c2.Close()
	c2.Expect("Activating state: ActiveState-CLI/Python3")

	c2.WaitForInput()
	c2.SendLine("exit")
	c2.ExpectExitCode(0)

}

func (suite *ActivateIntegrationTestSuite) TestActivate_JSON() {
	suite.testOutput("json")
}

func (suite *ActivateIntegrationTestSuite) TestActivate_EditorV0() {
	suite.testOutput("editor.v0")
}

func TestActivateIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ActivateIntegrationTestSuite))
}
