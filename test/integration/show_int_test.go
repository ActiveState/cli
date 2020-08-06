package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type ShowIntegrationTestSuite struct {
	suite.Suite
}

func (suite *ShowIntegrationTestSuite) TestShow() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("show")
	cp.Expect(`Name: Show`)
	cp.Expect(`Organization: cli-integration-tests`)
	cp.Expect(`Visibility: Public`)
	cp.Expect(`Latest Commit: d5d84598-fc2e-4a45-b075-a845e587b5bf`)
	cp.Expect(`Platforms: `)
	cp.Expect(` - CentOS`)
	cp.Expect(`Linux`)
	cp.Expect(`Languages: `)
	cp.Expect(` - python-3.6.6`)
	cp.Expect(`Events: `)
	cp.Expect(` - FIRST_INSTALL`)
	cp.Expect(` - AFTER_UPDATE`)
	cp.Expect(`Scripts: `)
	cp.Expect(` debug`)
	cp.Expect(` tests`)
	cp.ExpectExitCode(0)
}

func (suite *ShowIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := strings.TrimSpace(`
project: "https://platform.activestate.com/cli-integration-tests/Show?commitID=e8f3b07b-502f-4763-83c1-763b9b952e18"
constants:
  - name: DEBUG
    value: true
  - name: PYTHONPATH
    value: '%projectDir%/src:%projectDir%/tests'
    constraints:
        environment: dev,qa
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
}

func TestShowIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShowIntegrationTestSuite))
}
