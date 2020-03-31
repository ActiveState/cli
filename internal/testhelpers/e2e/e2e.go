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

func (s *Suite) TeardownTest() {
	if s.Session == nil {
		return
	}
	err := s.Session.Close()
	s.Require().NoError(err, "closing session")
}

func (s *Suite) Spawn(args ...string) *conproc.ConsoleProcess {
	return s.Session.Spawn(s.T(), args...)
}

func (s *Suite) SpawnCustom(cmdName string, args ...string) *conproc.ConsoleProcess {
	return s.Session.SpawnCustom(s.T(), cmdName, args...)
}

func (s *Suite) SpawnDirect(exe string, opts ...SpawnOptions) *conproc.ConsoleProcess {
	return s.Session.SpawnDirect(s.T(), exe, opts...)
}

func (s *Suite) PrepareActiveStateYAML(dir, contents string) {
	s.Session.PrepareActiveStateYAML(s.T(), dir, contents)
}

func (s *Suite) PrepareFile(path, contents string) {
	s.Session.PrepareFile(s.T(), path, contents)
}

func (s *Suite) LogoutUser() {
	s.Session.LogoutUser(s.T())
}
