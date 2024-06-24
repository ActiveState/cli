package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type PjFileIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PjFileIntegrationTestSuite) TestDeprecation() {
	suite.OnlyRunForTags(tagsuite.Projects)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(strings.TrimSpace(`
project: https://platform.activestate.com/ActiveState-CLI/test
platforms:
  - name: Linux64Label
languages:
  - name: Go
    constraints:
        platform: Windows10Label,Linux64Label
`))

	cp := ts.Spawn("scripts")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func TestPjFileIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PjFileIntegrationTestSuite))
}
