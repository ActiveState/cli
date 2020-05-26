package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type EditIntegrationTestSuite struct {
	suite.Suite
}

func (suite *EditIntegrationTestSuite) setup() (*e2e.Session, e2e.SpawnOptions) {
	ts := e2e.New(suite.T(), false)

	root := environment.GetRootPathUnsafe()
	editorScript := filepath.Join(root, "test", "integration", "assets", "editor", "main.go")

	target := filepath.Join(ts.Dirs.Work, "editor", "main.go")
	fail := fileutils.CopyFile(editorScript, target)
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
		e2e.WithArgs("build", "-o", "editor"+extension, target),
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
	cp.Expect("Script changes detected")
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
	cp.Wait()
}

func (suite *EditIntegrationTestSuite) TestEdit_UpdateCorrectPlatform() {
	ts, env := suite.setup()
	defer ts.Close()
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("scripts", "edit", "test-script"),
		e2e.WithWorkDirectory(ts.Dirs.Work),
		env,
	)
	cp.SendLine("Y")
	cp.ExpectExitCode(0)

	time.Sleep(time.Second * 2) // let CI env catch up

	project, fail := projectfile.FromPath(ts.Dirs.Work)
	suite.Require().NoError(fail.ToError())

	i := constraints.MostSpecificUnconstrained("test-script", project.Scripts.AsConstrainedEntities())
	suite.Require().True(i > -1, "Finds at least one script")
	suite.Contains(project.Scripts[i].Value, "more info!", "Output of edit command:\n%s", cp.Snapshot())
}

func TestEditIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(EditIntegrationTestSuite))
}
