package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type PushIntegrationTestSuite struct {
	integration.Suite
	username string
}

func (suite *PushIntegrationTestSuite) TestPush_AlreadyExists() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("push_editor_v0")
	defer cb()

	suite.LoginAsPersistentUser()
	username := "cli-integration-tests"
	namespace := fmt.Sprintf("%s/%s", username, "Python3")
	suite.Spawn(
		"init",
		namespace,
		"python3",
		"--path", filepath.Join(tempDir, namespace),
		"--skeleton", "editor",
	)
	suite.ExpectExitCode(0)
	suite.SetWd(filepath.Join(tempDir, namespace))
	suite.Spawn("push")
	suite.Wait()
	suite.Expect(fmt.Sprintf("The project %s/%s already exists", username, "Python3"))
}

func TestPushIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PushIntegrationTestSuite))
}
