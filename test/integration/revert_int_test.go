package integration

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type RevertIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *RevertIntegrationTestSuite) TestRevert() {
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/cli-integration-tests/Revert"`)

	cp := ts.Spawn("pull")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.ExpectExitCode(0)

	commitRe := regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)
	found := commitRe.FindString(cp.TrimmedSnapshot())
	fmt.Println(cp.TrimmedSnapshot())
	fmt.Println("Found:", found)

}

func TestRevertIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RevertIntegrationTestSuite))
}
