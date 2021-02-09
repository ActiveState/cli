package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	goversion "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type LanguagesIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_list() {
	suite.OnlyRunForTags(tagsuite.Languages)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	cp.Expect("3.6.6")
	cp.ExpectExitCode(0)
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_listNoCommitID() {
	suite.OnlyRunForTags(tagsuite.Languages)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAMLNoCommitID(ts)

	cp := ts.Spawn("languages")
	cp.ExpectNotExitCode(0)
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_install() {
	suite.OnlyRunForTags(tagsuite.Languages)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	username := ts.CreateNewUser()
	cp := ts.Spawn("auth", "--username", username, "--password", username)
	cp.Expect("You are logged in")
	cp.ExpectExitCode(0)
	cp.MatchState().TermState.StringBeforeCursor()

	path := cp.WorkDirectory()
	var err error
	if runtime.GOOS != "windows" {
		// On MacOS the tempdir is symlinked
		path, err = filepath.EvalSymlinks(cp.WorkDirectory())
		suite.Require().NoError(err)
	}

	cp = ts.Spawn("init", fmt.Sprintf("%s/%s", username, "Languages"), "python3", "--path", path)
	cp.ExpectLongString("successfully initialized")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("push")
	cp.Expect("Project created")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	cp.Expect("3.6.6")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("languages", "install", "python")
	cp.Expect("Language: python is already installed")
	cp.ExpectExitCode(1)

	cp = ts.Spawn("languages", "install", "python@3.8.2")
	cp.Expect("Language added: python@3.8.2")
	// This can take a little while
	cp.ExpectExitCode(0, 60*time.Second)

	cp = ts.Spawn("pull")
	cp.ExpectLongString("has been updated to the latest version available")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("languages")
	cp.Expect("Name")
	cp.Expect("Python")
	versionRe := regexp.MustCompile(`(\d+)\.(\d+).(\d+)`)
	cp.ExpectRe(versionRe.String())
	cp.ExpectExitCode(0)

	// assert that version number changed
	output := cp.MatchState().TermState.StringBeforeCursor()
	vs := versionRe.FindString(output)
	v, err := goversion.NewVersion(vs)
	suite.Require().NoError(err, "parsing version %s", vs)
	minVersion := goversion.Must(goversion.NewVersion("3.8.1"))
	suite.True(!v.LessThan(minVersion), "%v >= 3.8.1", v)
}

func (suite *LanguagesIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/Languages?commitID=e7df00bc-df4d-4e4a-97f7-efa741159bd2&branch=main"`
	ts.PrepareActiveStateYAML(asyData)
}

func (suite *LanguagesIntegrationTestSuite) PrepareActiveStateYAMLNoCommitID(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/Languages"`
	ts.PrepareActiveStateYAML(asyData)
}

func TestLanguagesIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(LanguagesIntegrationTestSuite))
}
