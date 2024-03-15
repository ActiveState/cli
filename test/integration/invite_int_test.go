package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type InviteIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InviteIntegrationTestSuite) TestInvite_NotAuthenticated() {
	suite.OnlyRunForTags(tagsuite.Invite)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Invite-Test", e2e.CommitIDNotChecked)

	cp := ts.Spawn("invite", "test-user@test.com")
	cp.Expect("You need to authenticate")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()
}

func TestInviteIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InviteIntegrationTestSuite))
}
