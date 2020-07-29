package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/termtest"
)

var (
	installAction   msiExecAction = "install"
	uninstallAction msiExecAction = "uninstall"
)

type msiExecAction string

func (a msiExecAction) cmd(msiPath, installPath string) string {
	msiAct := "/package"
	if a == uninstallAction {
		msiAct = "/uninstall"
	}

	pwshCmdForm := `$proc = Start-Process msiexec.exe -Wait -ArgumentList ` +
		`'%s %s /quiet /qn /norestart /log %s INSTALLDIR="%s"' -PassThru;` +
		`echo "exitcode:$($proc.ExitCode):";` + // use to ensure exit code is exactly 0
		`exit $proc.ExitCode;`
	pwshCmd := fmt.Sprintf(pwshCmdForm, msiAct, msiPath, a.logFileName(msiPath), installPath)
	return pwshCmd
}

func (a msiExecAction) logFileName(msiPath string) string {
	msiName := filepath.Base(strings.TrimSuffix(msiPath, filepath.Ext(msiPath)))
	return filepath.Join(environment.GetRootPathUnsafe(), fmt.Sprintf(`%s_%s.log`, msiName, string(a)))
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
