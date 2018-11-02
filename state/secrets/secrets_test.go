package secrets_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/stretchr/testify/suite"
)

type SecretCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *SecretCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	secretsClient := secretsapi.NewTestClient("http", constants.SecretsAPIHostTesting, constants.SecretsAPIPath, "bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient
}

func (suite *SecretCommandTestSuite) TestCommandConfig() {
	cmd, err := secrets.NewCommand(suite.secretsClient)
	suite.Require().NoError(err)

	conf := cmd.Config()
	suite.Equal("secrets", conf.Name)
	suite.Equal("secrets_cmd_description", conf.Description, "i18n symbol")

	subCmds := conf.GetCobraCmd().Commands()
	suite.Require().Len(subCmds, 1, "number of subcommands")
	suite.Equal("keypair", subCmds[0].Name())

	suite.Len(conf.Flags, 0, "number of command flags supported")
	suite.Len(conf.Arguments, 0, "number of commands args supported")
}

func (suite *SecretCommandTestSuite) Test_Execute_SucceedsWithoutArgs() {
	cmd, err := secrets.NewCommand(suite.secretsClient)
	suite.Require().NoError(err)
	cmd.Config().Execute()
	suite.NoError(failures.Handled(), "No failure occurred")
}

func Test_SecretCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretCommandTestSuite))
}
