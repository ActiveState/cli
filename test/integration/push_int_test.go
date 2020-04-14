package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type PushIntegrationTestSuite struct {
	suite.Suite
	username string
}

func (suite *PushIntegrationTestSuite) TestPush_AlreadyExists() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()
	username := "cli-integration-tests"
	namespace := fmt.Sprintf("%s/%s", username, "Python3")
	cp := ts.Spawn(
		"init",
		namespace,
		"python3",
		"--path", filepath.Join(ts.Dirs.Work, namespace),
		"--skeleton", "editor",
	)
	cp.ExpectExitCode(0)
	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.Expect(fmt.Sprintf("The project %s/%s already exists", username, "Python3"))
	cp.ExpectExitCode(0)
}

func TestPushIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PushIntegrationTestSuite))
}
