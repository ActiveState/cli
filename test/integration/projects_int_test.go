package integration

import (
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal-as/fileutils"
	"github.com/ActiveState/cli/internal-as/testhelpers/e2e"
	"github.com/ActiveState/cli/internal-as/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ProjectsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ProjectsIntegrationTestSuite) TestProjects() {
	suite.OnlyRunForTags(tagsuite.Projects)
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
}

func TestProjectsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsIntegrationTestSuite))
}
