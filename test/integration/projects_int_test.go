package integration

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ProjectsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ProjectsIntegrationTestSuite) TestProjects() {
	suite.OnlyRunForTags(tagsuite.Projects, tagsuite.VSCode)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/small-python"))
	cp.ExpectExitCode(0)
	cp = ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python3"))
	cp.ExpectExitCode(0)

	// Verify local checkouts and executables are grouped together under projects.
	cp = ts.SpawnWithOpts(e2e.WithArgs("projects"))
	cp.Expect("Python3")
	cp.Expect("Local Checkout")
	if runtime.GOOS != "windows" {
		cp.ExpectLongString(ts.Dirs.Work)
	} else {
		// Windows uses the long path here.
		longPath, _ := fileutils.GetLongPathName(ts.Dirs.Work)
		cp.ExpectLongString(longPath)
	}
	cp.Expect("Executables")
	cp.ExpectLongString(ts.Dirs.Cache)
	cp.Expect("small-python")
	cp.Expect("Local Checkout")
	if runtime.GOOS != "windows" {
		cp.ExpectLongString(ts.Dirs.Work)
	} else {
		// Windows uses the long path here.
		longPath, _ := fileutils.GetLongPathName(ts.Dirs.Work)
		cp.ExpectLongString(longPath)
	}
	cp.Expect("Executables")
	cp.ExpectLongString(ts.Dirs.Cache)
	cp.ExpectExitCode(0)

	// Verify separate "local_checkouts" and "executables" fields for JSON output.
	cp = ts.SpawnWithOpts(e2e.WithArgs("projects", "--output", "json"))
	cp.Expect(`"name":"Python3"`)
	cp.Expect(`"local_checkouts":["`)
	cp.ExpectLongString(filepath.Join(ts.Dirs.Work, "Python3") + `"]`)
	cp.Expect(`"executables":["`)
	cp.ExpectLongString(ts.Dirs.Cache)
	cp.ExpectExitCode(0)
}

func TestProjectsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsIntegrationTestSuite))
}
