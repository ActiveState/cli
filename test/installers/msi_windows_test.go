package installers_test

import (
	"fmt"
	"path/filepath"
	"regexp"
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

	checkPerlVersionCmd = "perl -v"
	checkPerlModulesCmd = "perldoc -l DBD::Pg"

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
		`echo "exitcode:$($proc.ExitCode):";` + // use to ensure exit code is exactly 0
		`refreshenv;` +
		`echo "path~path=$($Env:Path)~"`
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

func TestActivePerl(t *testing.T) {
	for _, msiFileName := range perlMsiFileNames {
		t.Run(msiFileName, func(t *testing.T) {
			m := newMsiFile(msiFileName)
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
