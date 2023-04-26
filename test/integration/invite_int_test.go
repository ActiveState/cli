package integration

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
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

	url := "https://platform.activestate.com/ActiveState-CLI/Invite-Test?branch=main&commitID=eb8dd176-d557-4adc-8b79-7b17e3a03bd7"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("invite", "test-user@test.com")
	cp.Expect("You need to authenticate")
	cp.ExpectNotExitCode(0)
}

func (suite *InviteIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Invite, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	url := "https://platform.activestate.com/ActiveState-CLI/Invite-Test?branch=main&commitID=eb8dd176-d557-4adc-8b79-7b17e3a03bd7"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("invite", "test-user@test.com", "-o", "json")
	cp.Expect(`{"errors":["You need to authenticate`)
	cp.ExpectNotExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestInviteIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InviteIntegrationTestSuite))
}
