package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type OrganizationsIntegrationTestSuite struct {
	integration.Suite
}

func (suite *OrganizationsIntegrationTestSuite) TestOrganizations_EditorV0() {
	suite.LoginAsPersistentUser()
	suite.Spawn("orgs", "--output", "editor.v0")
	suite.Wait()

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
	suite.Equal(fmt.Sprintf("[%s]", string(expected)), suite.TrimOutput())
}

func TestOrganizationsIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(OrganizationsIntegrationTestSuite))
}
