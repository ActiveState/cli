package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type LanguagesIntegrationTestSuite struct {
	suite.Suite
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_list() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	cp.Expect("3.6.6")
	cp.ExpectExitCode(0)
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_update() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	username := ts.CreateNewUser()
	cp := ts.Spawn("auth", "--username", username, "--password", username)
	cp.Expect("You are logged in")
	cp.ExpectExitCode(0)

	path := cp.WorkDirectory()
	var err error
	if runtime.GOOS != "windows" {
		// On MacOS the tempdir is symlinked
		path, err = filepath.EvalSymlinks(cp.WorkDirectory())
		suite.Require().NoError(err)
	}

	cp = ts.Spawn("init", fmt.Sprintf("%s/%s", username, "Languages"), "python3", "--path", path)
	cp.Expect("succesfully initialized")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("push")
	cp.Expect("Project created")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	cp.Expect("3.6.6")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("languages", "update", "python")
	// This can take a little while
	cp.ExpectExitCode(0, 30*time.Second)

	cp = ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	cp.Expect("3.8.1")
	cp.ExpectExitCode(0)
}

func (suite *LanguagesIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/Languages"`
	ts.PrepareActiveStateYAML(asyData)
}

func TestLanguagesIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(LanguagesIntegrationTestSuite))
}
