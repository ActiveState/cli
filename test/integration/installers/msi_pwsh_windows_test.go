package installers_test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/termtest"
)

var (
	installAction   msiExecAction = "install"
	uninstallAction msiExecAction = "uninstall"
)

type msiExecAction string

func (a msiExecAction) cmd(msiPath string) string {
	msiAct := "/package"
	if a == uninstallAction {
		msiAct = "/uninstall"
	}

	pwshCmdForm := `$proc = Start-Process msiexec.exe -Wait -ArgumentList ` +
		`"%s %s /quiet /qn /norestart /log %s" -PassThru;` +
		`$handle = $proc.Handle; $proc.WaitForExit();` +
		`echo "exitcode:$($proc.ExitCode):";` + // use to ensure exit code is exactly 0
		`refreshenv;` +
		`echo "path~path=C:\Perl64\bin;$Env:Path~";` // use for subsequent console processes
	pwshCmd := fmt.Sprintf(pwshCmdForm, msiAct, msiPath, a.logFileName(msiPath))
	return pwshCmd
}

func (a msiExecAction) logFileName(msiPath string) string {
	msiName := filepath.Base(strings.TrimSuffix(msiPath, filepath.Ext(msiPath)))
	return fmt.Sprintf(`%s\%s_%s.log`, logDir, msiName, string(a))
}

type pwshSession struct {
	*e2e.Session
}

func newPwshSession(t *testing.T) *pwshSession {
	return &pwshSession{e2e.New(t, false)}
}

func (s *pwshSession) Spawn(args ...string) *termtest.ConsoleProcess {
	return s.SpawnOpts(args)
}

func (s *pwshSession) SpawnOpts(args []string, opts ...e2e.SpawnOptions) *termtest.ConsoleProcess {
	as := append([]string{"/c"}, args...)
	opts = append(opts, e2e.WithArgs(as...))
	return s.Session.SpawnCmdWithOpts("powershell", opts...)
}

var (
	pathRegexp  = regexp.MustCompile("(?i).*?path~(path=.*?)~.*")
	pathReplace = "${1}"
)

func currentPath(cp *termtest.ConsoleProcess) string {
	pathInfo := cp.TrimmedSnapshot()
	return pathRegexp.ReplaceAllString(pathInfo, pathReplace)
}
