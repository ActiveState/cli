package installers_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/termtest"
)

var (
	rootDir = `Z:\bin`
	asToken = "ActiveState"

	perlMsiFileNames = []string{
		"ActivePerl-5.26.msi",
		//"ActivePerl-5.28.msi",
	}

	checkPerlCmd = "perl -v"

	installAction   msiExecAction = "install"
	uninstallAction msiExecAction = "uninstall"
)

type msiFile struct {
	path    string
	version string
}

func newMsiFile(filename string) *msiFile {
	return &msiFile{
		path:    filepath.Join(rootDir, filename),
		version: versionFromMsiFileName(filename),
	}
}

type msiExecAction string

func (a msiExecAction) cmd(msiPath string) string {
	msiAct := "/package"
	if a == uninstallAction {
		msiAct = "/uninstall"
	}

	pwshCmdForm := `$proc = Start-Process msiexec.exe -Wait -ArgumentList ` +
		`"%s %s /quiet /qn /norestart /log %s" -PassThru;` +
		`$handle = $proc.Handle; $proc.WaitForExit();` +
		`echo "exitcode:$($proc.ExitCode):";` // use to ensure exit code is exactly 0
	pwshCmd := fmt.Sprintf(pwshCmdForm, msiAct, msiPath, a.logFileName(msiPath))
	return pwshCmd
}

func (a msiExecAction) logFileName(msiPath string) string {
	msiName := filepath.Base(strings.TrimSuffix(msiPath, filepath.Ext(msiPath)))
	return fmt.Sprintf(`%s\%s_%s.log`, rootDir, msiName, string(a))
}

func versionFromMsiFileName(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	i := strings.LastIndexByte(name, '-')
	return name[i+1:]
}

func expectNone(cp *termtest.ConsoleProcess, fail func(...interface{}), values ...string) {
	trimmed := cp.TrimmedSnapshot()
	for _, val := range values {
		if strings.Contains(trimmed, val) {
			fail(fmt.Sprintf("incorrectly contains: %s", val))
		}
	}
}

type pwshSession struct {
	*e2e.Session
}

func newPwshSession(t *testing.T) *pwshSession {
	return &pwshSession{e2e.New(t, false)}
}

func (s *pwshSession) Spawn(args ...string) *termtest.ConsoleProcess {
	as := []string{"/c"}
	as = append(as, args...)
	return s.Session.SpawnCmd("powershell", args...)
}

func TestActivePerl(t *testing.T) {
	for _, msiFileName := range perlMsiFileNames {
		t.Run(msiFileName, func(t *testing.T) {
			m := newMsiFile(msiFileName)
			s := newPwshSession(t)

			cp := s.Spawn(installAction.cmd(m.path))
			cp.Expect("exitcode:0:", time.Minute*3)
			cp.ExpectExitCode(0)

			cp = s.Spawn(checkPerlCmd)
			cp.Expect(m.version)
			cp.Expect(asToken)
			cp.ExpectExitCode(0)

			cp = s.Spawn(uninstallAction.cmd(m.path))
			cp.Expect("exitcode:0:", time.Minute)
			cp.ExpectExitCode(0)

			cp = s.Spawn(checkPerlCmd)
			cp.Expect("'perl' is not recognized")
			cp.ExpectNotExitCode(0)
		})
	}
}
