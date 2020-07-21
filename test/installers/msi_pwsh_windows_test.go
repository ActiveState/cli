package installers_test

import (
	"regexp"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/termtest"
)

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
