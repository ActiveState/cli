package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PushIntegrationTestSuite struct {
	suite.Suite
	username string
}

func (suite *PushIntegrationTestSuite) TestPush_EditorV0() {
	username := suite.CreateNewUser()

	namespace := fmt.Sprintf("%s/%s", username, "Python3")
	cp := suite.Spawn(
		"init",
		namespace,
		"python3",
		"--path", filepath.Join(suite.WorkDirectory(), namespace),
		"--skeleton", "editor",
	)
	defer cp.Close()
	cp.ExpectExitCode(0)
	suite.SetWd(filepath.Join(tempDir, namespace))
	suite.Spawn("push")
	suite.Expect(fmt.Sprintf("Creating project Python3 under %s", username))
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
