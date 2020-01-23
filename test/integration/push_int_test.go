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

func (suite *PushIntegrationTestSuite) TestPush_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("push_editor_v0")
	defer cb()

	username := suite.CreateNewUser()

	namespace := fmt.Sprintf("%s/%s", username, "Python3")
	suite.Spawn(
		"init",
		namespace,
		"--language", "python3",
		"--path", filepath.Join(tempDir, namespace),
		"--skeleton", "editor",
	)
	suite.Wait()
	suite.SetWd(filepath.Join(tempDir, namespace))
	suite.Spawn("push")
	suite.Expect(fmt.Sprintf("Creating project Python3 under %s", username))
}

func TestPushIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PushIntegrationTestSuite))
}
