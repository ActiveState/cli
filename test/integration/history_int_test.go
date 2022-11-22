package integration

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type HistoryIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *HistoryIntegrationTestSuite) TestHistory_History() {
	suite.OnlyRunForTags(tagsuite.History)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("checkout", "ActiveState-CLI/History")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("history"),
		e2e.WithWorkDirectory(filepath.Join(ts.Dirs.Work, "History")),
	)
	cp.Expect("Commit")
	cp.Expect("Author")
	cp.Expect("Date")
	cp.Expect("Message")
	cp.ExpectLongString("• requests (2.26.0 → 2.7.0)")
	cp.ExpectLongString("• autopip (1.6.0 → Auto)")
	cp.Expect("+ autopip 1.6.0")
	cp.Expect("- convertdate")
	cp.Expect(`+ Platform`)
	cp.ExpectExitCode(0)
}

func (suite *HistoryIntegrationTestSuite) TestRevert() {
	suite.OnlyRunForTags(tagsuite.History)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	username := ts.CreateNewUser()
	namespace := fmt.Sprintf("%s/%s", username, "revert-a")

	createRevertProject(ts, namespace, true)

	projFile := ts.CurrentProjectFile()

	namespace = fmt.Sprintf("%s/%s", username, "revert-b")
	createRevertProject(ts, namespace, false)

	cp := ts.Spawn("revert", projFile.CommitID())
	cp.ExpectExitCode(1)
}

func createRevertProject(ts *e2e.Session, namespace string, withUninstall bool) {
	cp := ts.Spawn("init", namespace, "python3", "--path="+ts.Dirs.Work, "--skeleton=editor")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("shell")

	cp = ts.Spawn("install", "json2")
	cp.ExpectRe("(?:Package added|being built)", 30*time.Second)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "dateparser@0.7.2")
	cp.ExpectRe("(?:Package added|being built)", 30*time.Second)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("push")
	cp.ExpectExitCode(0)

	if withUninstall {
		cp = ts.Spawn("remove", "json2")
		cp.ExpectRe("(?:Package removed|being built)", 30*time.Second)
		cp.ExpectExitCode(0)
	}
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
