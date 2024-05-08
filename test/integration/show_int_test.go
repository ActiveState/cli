package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ShowIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ShowIntegrationTestSuite) TestShow() {
	suite.OnlyRunForTags(tagsuite.Show)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareProject(ts)

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectInput(e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.Spawn("show")
	cp.Expect(`Name`)
	cp.Expect(`Show`)
	cp.Expect(`Organization`)
	cp.Expect(`cli-integration-tests`)
	cp.Expect(`Namespace`)
	cp.Expect(`cli-integration-tests/Show`)
	cp.Expect(`Location`)
	cp.Expect(ts.Dirs.Work)
	cp.Expect(`Executables`)
	cp.Expect(ts.Dirs.Cache)
	cp.Expect(`Visibility`)
	cp.Expect(`Public`)
	cp.Expect(`Latest Commit`)
	cp.Expect(`d5d84598-fc2e-4a45-b075-a845e587b5bf`)
	cp.Expect(`Events`)
	cp.Expect(`• FIRST_INSTALL`)
	cp.Expect(`• AFTER_UPDATE`)
	cp.Expect(`Scripts`)
	cp.Expect(`• debug`)
	cp.Expect(`Platforms`)
	cp.Expect(`CentOS`)
	cp.Expect(`Languages`)
	cp.Expect(`python`)
	cp.Expect(`3.6.6`)
	cp.ExpectExitCode(0)
}

func (suite *ShowIntegrationTestSuite) TestShowWithoutBranch() {
	suite.OnlyRunForTags(tagsuite.Show, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Show", "e8f3b07b-502f-4763-83c1-763b9b952e18")

	cp := ts.SpawnWithOpts(e2e.OptArgs("show"))
	cp.ExpectExitCode(0)

	contents, err := fileutils.ReadFile(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Require().NoError(err)

	suite.Contains(string(contents), "branch="+constants.DefaultBranchName)
}

func (suite *ShowIntegrationTestSuite) PrepareProject(ts *e2e.Session) {
	asyData := strings.TrimSpace(`
project: "https://platform.activestate.com/cli-integration-tests/Show?branch=main"
constants:
  - name: DEBUG
    value: true
  - name: PYTHONPATH
    value: '%projectDir%/src:%projectDir%/tests'
  - name: PYTHONPATH
    value: '%projectDir%/src:%projectDir%/tests'
events:
  - name: FIRST_INSTALL
    value: '%pythonExe% %projectDir%/setup.py prepare'
  - name: AFTER_UPDATE
    value: '%pythonExe% %projectDir%/setup.py prepare'
scripts:
  - name: tests
    value: pytest %projectDir%/tests
  - name: debug
    value: debug foo
`)

	ts.PrepareActiveStateYAML(asyData)
	ts.PrepareCommitIdFile("d5d84598-fc2e-4a45-b075-a845e587b5bf")
}

func (suite *ShowIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Show, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/small-python", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("show", "-o", "json")
	cp.Expect(`"project_url":`)
	cp.Expect(`"name":`)
	cp.Expect(`"platforms":`)
	cp.Expect(`"languages":`)
	cp.Expect(`"secrets":`)
	cp.Expect(`"events":`)
	cp.Expect(`"scripts":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestShowIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ShowIntegrationTestSuite))
}
