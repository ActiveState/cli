package e2e

import (
	"github.com/ActiveState/cli/pkg/expect/conproc"
	"github.com/stretchr/testify/suite"
)

// Suite is a test suite that encapsulates an end-to-end session
// This can be used as a convenient construct to set-up a session for every test
// in a suite.
// But beware: It might not be thread-safe...  It is probably safer to use session directly
type Suite struct {
	suite.Suite
	Session *Session
}

func (s *Suite) SetupTest() {
	s.Session = New(s.T(), false)
}

func (s *Suite) TearDownTest() {
	if s.Session == nil {
		return
	}
	err := s.Session.Close()
	s.Require().NoError(err, "closing session")
}

func (s *Suite) Spawn(args ...string) *conproc.ConsoleProcess {
	return s.Session.Spawn(args...)
}

func (s *Suite) SpawnWithOpts(opts ...SpawnOptions) *conproc.ConsoleProcess {
	return s.Session.SpawnWithOpts(opts...)
}

func (s *Suite) SpawnCustom(cmdName string, args ...string) *conproc.ConsoleProcess {
	return s.Session.SpawnCustom(cmdName, args...)
}

func (s *Suite) SpawnCustomWithOpts(exe string, opts ...SpawnOptions) *conproc.ConsoleProcess {
	return s.Session.SpawnCustomWithOpts(exe, opts...)
}

func (s *Suite) PrepareActiveStateYAML(contents string) {
	s.Session.PrepareActiveStateYAML(contents)
}

func (s *Suite) PrepareFile(path, contents string) {
	s.Session.PrepareFile(path, contents)
}

func (s *Suite) CreateNewUser() string {
	return s.Session.CreateNewUser()
}

func (s *Suite) LoginAsPersistentUser() {
	s.Session.LoginAsPersistentUser()
}

func (s *Suite) LoginUser(userName string) {
	s.Session.LoginUser(userName)
}

func (s *Suite) LogoutUser() {
	s.Session.LogoutUser()
}

func (s *Suite) WorkDirectory() string {
	return s.Session.Dirs.Work
}
