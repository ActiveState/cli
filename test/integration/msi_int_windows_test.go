package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/autarch/testify/assert"
	"github.com/autarch/testify/require"
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

func assertRegistryPathInclues(t *testing.T, path string) {
	out, err := exec.Command("reg", "query", `HKLM\SYSTEM\ControlSet001\Control\Session Manager\Environment`, "/v", "Path").Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), path, "Windows system PATH should contain our target dir")
}

func addPathToEnv(path string) string {
	oldPath, ok := os.LookupEnv("PATH")
	if !ok {
		oldPath = ""
	}
	return fmt.Sprintf("PATH=%s;%s", path, oldPath)
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

	installPath := `C:\Perl64 with spaces`

	for _, msiFilePath := range perlMsiFilePaths {
		t.Run(filepath.Base(msiFilePath), func(t *testing.T) {
			m := newMsiFile(msiFilePath)
			s := newPwshSession(t)

			cp := s.Spawn(installAction.cmd(m.path, installPath))
			cp.Expect("exitcode:0:", time.Minute*3)
			cp.ExpectExitCode(0)

			assertRegistryPathInclues(t, installPath)

			pathEnv := addPathToEnv(filepath.Join(installPath, "bin"))
			checkPerlArgs := []string{checkPerlVersionCmd}
			cp = s.SpawnOpts(checkPerlArgs, e2e.AppendEnv(pathEnv))
			cp.Expect(m.version)
			cp.Expect(asToken)
			cp.ExpectExitCode(0)

			checkPerlModsArgs := []string{checkPerlModulesCmd}
			cp = s.SpawnOpts(checkPerlModsArgs, e2e.AppendEnv(pathEnv))
			cp.Expect("Pg.pm")
			cp.ExpectExitCode(0)

			cp = s.Spawn(uninstallAction.cmd(m.path, installPath))
			cp.Expect("exitcode:0:", time.Minute)
			cp.ExpectExitCode(0)

			cp = s.SpawnOpts(checkPerlArgs, e2e.AppendEnv(pathEnv))
			cp.Expect("'perl' is not recognized")
			cp.ExpectNotExitCode(0)
		})
	}
}
