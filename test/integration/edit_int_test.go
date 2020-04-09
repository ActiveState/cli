package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type EditIntegrationTestSuite struct {
	suite.Suite
}

func (suite *EditIntegrationTestSuite) setup() (*e2e.Session, e2e.SpawnOptions) {
	ts := e2e.New(suite.T(), false)

	root := environment.GetRootPathUnsafe()
	editorScript := filepath.Join(root, "test", "integration", "assets", "editor", "main.go")

	fail := fileutils.CopyFile(editorScript, filepath.Join(ts.Dirs.Work, "editor", "main.go"))
	suite.Require().NoError(fail.ToError())

	configFileContent := strings.TrimSpace(`
project: "https://platform.activestate.com/EditOrg/EditProject?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: test-script
    value: echo "hello test"
    constraints:
      os: macos,linux
  - name: test-script
    value: echo hello test
    constraints:
      os: windows
`)
	ts.PrepareActiveStateYAML(configFileContent)

	editorScriptDir := filepath.Join(ts.Dirs.Work, "editor")

	var extension string
	if runtime.GOOS == "windows" {
		extension = ".exe"
	}
	cp := ts.SpawnCmdWithOpts(
		"go",
		e2e.WithArgs("build", "-o", "editor"+extension),
		e2e.WithWorkDirectory(editorScriptDir),
	)
	cp.ExpectExitCode(0)

	suite.Require().FileExists(filepath.Join(editorScriptDir, "editor"+extension))
	return ts, e2e.AppendEnv(fmt.Sprintf("EDITOR=%s", filepath.Join(editorScriptDir, "editor"+extension)))
}

func (suite *EditIntegrationTestSuite) TearDownTest() {
	projectfile.Reset()
}

func (suite *EditIntegrationTestSuite) TestEdit() {
	ts, env := suite.setup()
	defer ts.Close()
	cp := ts.SpawnWithOpts(e2e.WithArgs("scripts", "edit", "test-script"), env)
	cp.Expect("Watching file changes")
	cp.Expect("Are you done editing?")
	// Can't consistently get this line detected on CI
	// suite.Expect("Script changes detected")
	cp.SendLine("Y")
	cp.ExpectExitCode(0)
}

func (suite *EditIntegrationTestSuite) TestEdit_NonInteractive() {
	ts, env := suite.setup()
	defer ts.Close()
	extraEnv := e2e.AppendEnv("ACTIVESTATE_NONINTERACTIVE=true")

	cp := ts.SpawnWithOpts(e2e.WithArgs("scripts", "edit", "test-script"), env, extraEnv)
	cp.Expect("Watching file changes")
	// Can't consistently get this line detected on CI
	cp.Expect("Script changes detected")
	cp.SendCtrlC()
	cp.ExpectExitCode(0)
}

func (suite *EditIntegrationTestSuite) TestEdit_UpdateCorrectPlatform() {
	ts, env := suite.setup()
	defer ts.Close()
	cp := ts.SpawnWithOpts(e2e.WithArgs("scripts", "edit", "test-script"), env)
	cp.SendLine("Y")
	cp.ExpectExitCode(0)

	time.Sleep(time.Second * 2) // let CI env catch up

	project := projectfile.Get()
	for _, script := range project.Scripts {
		if script.Name == "test-script" {
			if !constraints.IsConstrained(script.Constraints) {
				suite.Contains(script.Value, "more info!")
			}
		}
	}
}

func TestEditIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(EditIntegrationTestSuite))
}
