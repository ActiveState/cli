package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

var (
	msiDir = mustFilepathByProjectRoot(`/build/msi`)
	logDir = mustFilepathByProjectRoot(`/build`)

	msiExt        = ".msi"
	perlMsiPrefix = "ActivePerl"

	asToken = "ActiveState"

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

func msiFilePaths(dir, prefix string) ([]string, error) {
	var filePaths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		curBase := filepath.Base(path)
		curExt := filepath.Ext(curBase)

		if strings.HasPrefix(curBase, perlMsiPrefix) && curExt == msiExt {
			filePaths = append(filePaths, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return filePaths, nil
}

func TestActivePerl(t *testing.T) {
	if !e2e.RunningOnCI() && false {
		t.Skipf("Skipping; Not running on CI")
	}

	perlMsiFilePaths, err := msiFilePaths(msiDir, perlMsiPrefix)
	if err != nil {
		t.Fatal(err)
	}

	if len(perlMsiFilePaths) == 0 {
		t.Fatalf("no %q msi files found in %q", perlMsiPrefix, msiDir)
	}

	for _, msiFilePath := range perlMsiFilePaths {
		t.Run(filepath.Base(msiFilePath), func(t *testing.T) {
			m := newMsiFile(msiFilePath)
			s := newPwshSession(t)

			cp := s.Spawn(installAction.cmd(m.path))
			cp.Expect("exitcode:0:", time.Minute*3)
			cp.ExpectExitCode(0)
			path := currentPath(cp)

			checkPerlArgs := []string{checkPerlVersionCmd}
			cp = s.SpawnOpts(checkPerlArgs, e2e.AppendEnv(path))
			cp.Expect(m.version)
			cp.Expect(asToken)
			cp.ExpectExitCode(0)

			checkPerlModsArgs := []string{checkPerlModulesCmd}
			cp = s.SpawnOpts(checkPerlModsArgs, e2e.AppendEnv(path))
			cp.Expect("Pg.pm")
			cp.ExpectExitCode(0)

			cp = s.Spawn(uninstallAction.cmd(m.path))
			cp.Expect("exitcode:0:", time.Minute)
			cp.ExpectExitCode(0)
			path = currentPath(cp)

			cp = s.SpawnOpts(checkPerlArgs, e2e.AppendEnv(path))
			cp.Expect("'perl' is not recognized")
			cp.ExpectNotExitCode(0)
		})
	}
}
