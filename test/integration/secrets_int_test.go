package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type SecretsIntegrationTestSuite struct {
	integration.Suite
}

func (suite *SecretsIntegrationTestSuite) TestSecrets_EditorV0() {

}

func TestSecretsIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(SecretsIntegrationTestSuite))
}
