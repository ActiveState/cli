package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
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
	res := cp.MatchState().TermState.StringBeforeCursor()
	fmt.Println(res)

	path := cp.WorkDirectory()
	var err error
	if runtime.GOOS != "windows" {
		// On MacOS the tempdir is symlinked
		path, err = filepath.EvalSymlinks(cp.WorkDirectory())
		suite.Require().NoError(err)
	}

	cp = ts.Spawn("init", fmt.Sprintf("%s/%s", username, "Languages"), "python3", "--path", path)
	cp.Expect("successfully initialized")
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
	cp.ExpectExitCode(0, 60*time.Second)

	cp = ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	versionRe := regexp.MustCompile(`(\d+)\.(\d+).(\d+)`)
	cp.ExpectRe(versionRe.String())
	cp.ExpectExitCode(0)

	// assert that version number increased at least 3.8.1
	output := cp.MatchState().TermState.StringBeforeCursor()
	vs := versionRe.FindString(output)
	v, err := semver.Parse(vs)
	suite.Require().NoError(err, "parsing version %s", vs)
	minVersion := semver.MustParse("3.8.1")
	suite.True(v.GTE(minVersion), "%v >= 3.8.1", v)
}

func (suite *LanguagesIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/Languages"`
	ts.PrepareActiveStateYAML(asyData)
}

func TestLanguagesIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(LanguagesIntegrationTestSuite))
}
