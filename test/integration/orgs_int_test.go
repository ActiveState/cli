package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type OrganizationsIntegrationTestSuite struct {
	suite.Suite
}

func (suite *OrganizationsIntegrationTestSuite) TestOrganizations_EditorV0() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("orgs", "--output", "editor.v0")
	cp.ExpectExitCode(0)

	org := struct {
		Name            string `json:"name,omitempty"`
		URLName         string `json:"URLName,omitempty"`
		Tier            string `json:"tier,omitempty"`
		PrivateProjects bool   `json:"privateProjects"`
	}{
		"Test-Organization",
		"Test-Organization",
		"free",
		false,
	}

	expected, err := json.Marshal(org)
	suite.Require().NoError(err)

	suite.Equal(fmt.Sprintf("[%s]", string(expected)), cp.TrimmedSnapshot())
}

func TestOrganizationsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsIntegrationTestSuite))
}
