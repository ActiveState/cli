package integration

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/require"
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

func (suite *HistoryIntegrationTestSuite) TestRevert_failsOnCommitNotInHistory() {
	suite.OnlyRunForTags(tagsuite.History)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	username := ts.CreateNewUser()

	namespaceA := fmt.Sprintf("%s/%s", username, "revert-a")
	wdA := createRevertProject(ts, namespaceA, true)

	projFile, err := projectfile.FromPath(filepath.Join(wdA, "activestate.yaml"))
	require.NoError(suite.T(), err, "cannot get current projectfile")

	namespaceB := fmt.Sprintf("%s/%s", username, "revert-b")
	wdB := createRevertProject(ts, namespaceB, false)

	cp := ts.SpawnWithOpts(e2e.WithArgs("revert", projFile.CommitID()), e2e.WithWorkDirectory(wdB))
	cp.SendLine("Y")
	cp.Expect(projFile.CommitID())
	cp.Expect("The target commit is not")
	cp.ExpectNotExitCode(0)
}

func createRevertProject(ts *e2e.Session, namespace string, withUninstall bool) (workingDir string) {
	wd := filepath.Join(ts.Dirs.Work, namespace)
	cp := ts.Spawn("init", namespace, "python3", "--path="+wd, "--skeleton=editor")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("install", "json2"), e2e.WithWorkDirectory(wd))
	cp.ExpectRe("(?:Package added|being built)", 30*time.Second)
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("install", "dateparser@0.7.2"), e2e.WithWorkDirectory(wd))
	cp.ExpectRe("(?:Package added|being built)", 30*time.Second)
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("push", "--non-interactive"), e2e.WithWorkDirectory(wd))
	cp.ExpectExitCode(0)

	if withUninstall {
		cp = ts.SpawnWithOpts(e2e.WithArgs("uninstall", "json2"), e2e.WithWorkDirectory(wd))
		cp.ExpectRe("(?:Package uninstalled|being built)", 30*time.Second)
		cp.ExpectExitCode(0)
	}

	return wd
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
