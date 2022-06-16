package automation

import (
	"fmt"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
	"path/filepath"
	"testing"
	"time"
)

type SearchAutomationTestSuite struct {
	tagsuite.Suite
}

func (suite *SearchAutomationTestSuite) TestSearch_NoArg() {
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
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search", "--language")
	cp.ExpectLongString("flag needs an argument: --language")
	cp.ExpectExitCode(1)

}

func (suite *SearchAutomationTestSuite) TestSearch_OutProject() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search", "flask")
	cp.ExpectLongString("Language must be provided by flag or by running this command within a project.")
	cp.ExpectExitCode(1)

}

func (suite *SearchAutomationTestSuite) TestSearch_Flask() {
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
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search", "--language", "python", "flask")
	cp.Expect("flaskcap")
	cp.Expect("Flask-CDN")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("search", "--language", "ruby", "ao")
	cp.Expect("kaomoji")
	cp.Expect("mao")
	cp.ExpectExitCode(0)

	url := "https://platform.activestate.com/qamainorg/public?branch=main&commitID=32e543ee-b6ab-4f59-9b28-ad830ec6980e"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp = ts.Spawn("search", "--language", "python", "flask")
	cp.Expect("flaskcap")
	cp.Expect("Flask-CDN")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("search", "--language", "ruby", "ao")
	cp.Expect("kaomoji")
	cp.Expect("mao")
	cp.ExpectExitCode(0)
	cp.Wait(5 * time.Second)
	fmt.Println(cp.Snapshot())

}

func (suite *SearchAutomationTestSuite) TestSearch_ExactTermFlag() {
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
