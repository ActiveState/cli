package automation

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
	"github.com/stretchr/testify/suite"
)

type SearchAutomationTestSuite struct {
	testsuite.Suite
}

func (suite *SearchAutomationTestSuite) TestSearch_NoArg() {
	suite.OnlyRunForTags(testsuite.TagAutomation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search")
	cp.ExpectLongString("The following argument is required:")
	cp.ExpectExitCode(1)

	cp = ts.Spawn("search", "--language", "python")
	cp.ExpectLongString("The following argument is required:")
	cp.ExpectExitCode(1)
}

func (suite *SearchAutomationTestSuite) TestSearch_NoLanguageArg() {
	suite.OnlyRunForTags(testsuite.TagAutomation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search", "--language")
	cp.ExpectLongString("Flag needs an argument: --language")
	cp.ExpectExitCode(1)
}

func (suite *SearchAutomationTestSuite) TestSearch_OutProject() {
	suite.OnlyRunForTags(testsuite.TagAutomation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search", "flask")
	cp.ExpectLongString("Language must be provided by flag")
	cp.ExpectExitCode(1)
}

func (suite *SearchAutomationTestSuite) TestSearch_Flask() {
	suite.OnlyRunForTags(testsuite.TagAutomation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	url := "https://platform.activestate.com/qamainorg/public?branch=main&commitID=32e543ee-b6ab-4f59-9b28-ad830ec6980e"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("search", "flask")
	cp.Expect("flaskcap")
	cp.Expect("Flask-CDN")
	cp.ExpectExitCode(0)
}

func (suite *SearchAutomationTestSuite) TestSearch_LanguageFlag() {
	suite.OnlyRunForTags(testsuite.TagAutomation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search", "--language", "python", "flask")
	cp.Expect("flaskcap")
	cp.Expect("Flask-CDN")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("search", "--language", "ruby", "radar")
	cp.Expect("radar-api")
	cp.Expect("radar-app")
	cp.ExpectExitCode(0)

	url := "https://platform.activestate.com/qamainorg/public?branch=main&commitID=32e543ee-b6ab-4f59-9b28-ad830ec6980e"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp = ts.Spawn("search", "--language", "python", "flask")
	cp.Expect("flaskcap")
	cp.Expect("Flask-CDN")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("search", "--language", "ruby", "radar")
	cp.Expect("radar-api")
	cp.Expect("radar-app")
	cp.ExpectExitCode(0)
}

func (suite *SearchAutomationTestSuite) TestSearch_ExactTermFlag() {
	suite.OnlyRunForTags(testsuite.TagAutomation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search", "--language", "python", "--exact-term", "flask")
	cp.Expect("Latest Version")
	cp.Expect("Flask")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("search", "--language", "ruby", "--exact-term", "ao")
	cp.Expect("Latest Version")
	cp.Expect("ao")
	cp.ExpectExitCode(0)

	url := "https://platform.activestate.com/qamainorg/public?branch=main&commitID=32e543ee-b6ab-4f59-9b28-ad830ec6980e"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp = ts.Spawn("search", "--language", "python", "--exact-term", "flask")
	cp.Expect("Latest Version")
	cp.Expect("Flask")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("search", "--language", "ruby", "--exact-term", "ao")
	cp.Expect("Latest Version")
	cp.Expect("ao")
	cp.ExpectExitCode(0)
}

func TestSearchAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(SearchAutomationTestSuite))
}
