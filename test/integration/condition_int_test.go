package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

type ConditionIntegrationTestSuite struct {
	suite.Suite
}

func (suite *ConditionIntegrationTestSuite) TestCondition() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("run", "test")
	cp.Expect(`projectNameValue`)
	cp.Expect(`projectOwnerValue`)
	cp.Expect(`projectNamespaceValue`)
	cp.Expect(`osNameValue`)
	cp.Expect(`osVersionValue`)
	cp.Expect(`osArchValue`)
	cp.Expect(`shellValue`)
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
	)
	cp.Expect(`Activation Event Ran`)
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ConditionIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := strings.TrimSpace(`
project: https://platform.activestate.com/ActiveState-CLI/test?commitID=9090c128-e948-4388-8f7f-96e2c1e00d98
constants:
  - name: projectName
    value: invalidProjectName
    if: false
  - name: projectName
    value: projectNameValue
    if: ne .Project.Name ""
  - name: projectName
    value: invalidProjectName
    if: false
  - name: projectOwner
    value: projectOwnerValue
    if: ne .Project.Owner ""
  - name: projectNamespace
    value: projectNamespaceValue
    if: ne .Project.Namespace ""
  - name: osName
    value: osNameValue
    if: ne .OS.Name ""
  - name: osVersion
    value: osVersionValue
    if: ne .OS.Version.Name ""
  - name: osArch
    value: osArchValue
    if: ne .OS.Architecture ""
  - name: shell
    value: shellValue
    if: ne .Shell ""
scripts:
  - name: test
    value: echo wrong script
    if: false
  - name: test
    standalone: true
    language: bash
    value: |
      echo ${constants.projectName}
      echo ${constants.projectOwner}
      echo ${constants.projectNamespace}
      echo ${constants.osName}
      echo ${constants.osVersion}
      echo ${constants.osArch}
      echo ${constants.shell}
    if: ne .Shell ""
  - name: test
    value: echo wrong script
    if: false
events:
  - name: ACTIVATE
    value: echo "Wrong event"
    if: false
  - name: ACTIVATE
    value: echo "Activation Event Ran"
    if: ne .Shell ""
  - name: ACTIVATE
    value: echo "Wrong event"
    if: false
`)

	ts.PrepareActiveStateYAML(asyData)
}

func TestConditionIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ConditionIntegrationTestSuite))
}
