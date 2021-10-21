package integration

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type LanguageMSITestSuite struct {
	tagsuite.Suite
}

var (
	msiDir = mustFilepathByProjectRoot(`/build/msi`)
	logDir = mustFilepathByProjectRoot(`/build`)

	checkPerlVersionCmd = "perl -v"
	checkPerlModulesCmd = "perldoc -l DBD::Pg"
)

func mustFilepathByProjectRoot(path string) string {
	root := environment.GetRootPathUnsafe()
	return filepath.Join(root, path)
}

type msiFile struct {
	path    string
	version string
}

func newMsiFile(filePath string) *msiFile {
	return &msiFile{
		path:    filePath,
		version: versionFromMsiFileName(filePath),
	}
}

func versionFromMsiFileName(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	i := strings.LastIndexByte(name, '-')
	return name[i+1:]
}

func (suite *LanguageMSITestSuite) assertRegistryPathIncludes(path string) {
	out, err := exec.Command("reg", "query", `HKLM\SYSTEM\ControlSet001\Control\Session Manager\Environment`, "/v", "Path").Output()
	suite.Require().NoError(err)
	suite.Assert().Contains(string(out), path, "Windows system PATH should contain our target dir")
}

func (suite *LanguageMSITestSuite) TestActivePerl() {
	suite.OnlyRunForTags(tagsuite.MSI, tagsuite.Perl)
	if !e2e.RunningOnCI() && false {
		suite.T().Skipf("Skipping; Not running on CI")
	}

	perlMsiFilePaths := []string{
		filepath.Join(msiDir, "ActivePerl-5.26.msi"),
		filepath.Join(msiDir, "ActivePerl-5.28.msi"),
	}

	installPath := `C:\Perl64 with spaces`

	for _, msiFilePath := range perlMsiFilePaths {
		suite.Run(filepath.Base(msiFilePath), func() {
			m := newMsiFile(msiFilePath)
			s := newPwshSession(suite.T())

			cp := s.Spawn(installAction.cmd(m.path, installPath))
			cp.Expect("exitcode:0:", time.Minute*3)
			cp.ExpectExitCode(0)

			suite.assertRegistryPathIncludes(installPath)

			pathEnv := fmt.Sprintf("PATH=%s", filepath.Join(installPath, "bin"))
			cp = s.SpawnOpts("perl -v", e2e.AppendEnv(pathEnv))
			cp.Expect(m.version)
			cp.Expect("ActiveState")
			cp.ExpectExitCode(0)

			cp = s.SpawnOpts("perldoc -l DBD::Pg", e2e.AppendEnv(pathEnv))
			cp.Expect("Pg.pm")
			cp.ExpectExitCode(0)

			cp = s.Spawn(uninstallAction.cmd(m.path, installPath))
			cp.Expect("exitcode:0:", time.Minute)
			cp.ExpectExitCode(0)

			cp = s.SpawnOpts("perl -v", e2e.AppendEnv(pathEnv))
			cp.Expect("'perl' is not recognized")
			cp.ExpectNotExitCode(0)
		})
	}
}

func TestLanguageMSITestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LanguageMSITestSuite))
}
