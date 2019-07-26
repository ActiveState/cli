package expect

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

type Suite struct {
	suite.Suite
	process      *Process
	processEnded chan bool
	env          []string
	Executable   string
}

func (s *Suite) ClearEnv() {
	s.env = []string{}
}

func (s *Suite) AppendEnv(env []string) {
	s.env = append(s.env, env...)
}

func (s *Suite) Spawn(args ...string) {
	s.SpawnCustom(s.Executable, args...)
}

func (s *Suite) SpawnCustom(executable string, args ...string) {
	wd, _ := os.Getwd()
	commandLine := fmt.Sprintf("%s %s", executable, strings.Join(args, " "))
	fmt.Printf("Spawning '%s' from %s\n", commandLine, wd)
	s.process = NewProcess(executable, args...)
	s.process.SetEnv(s.env)
	s.processEnded = make(chan bool)

	stack := stacktrace.Get()

	go func() {
		time.Sleep(10 * time.Millisecond) // Ensure we don't start receiving output before the expect rule has been set
		err := s.process.Run()
		if err != nil {
			s.FailNow("Error while running process", "error: %v, output:\n---\n%s\n---\nstack:\n%s\n",
				err, s.process.CombinedOutput(), stack.String())
		}
		s.processEnded <- true
	}()
}

func (s *Suite) ExitCode() int {
	return s.process.ExitCode()
}

func (s *Suite) WaitForInput(timeout ...time.Duration) {
	usr, err := user.Current()
	s.Require().NoError(err)

	msg := "echo wait_ready_$HOME"
	if runtime.GOOS == "windows" {
		msg = "echo wait_ready_%USERPROFILE%"
	}

	s.Send(msg)
	s.Expect("wait_ready_"+usr.HomeDir, timeout...)
}

func (s *Suite) Wait(timeout ...time.Duration) {
	t := 10 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}

	select {
	case <-s.processEnded:
		return
	case <-time.After(t):
		s.FailNow("Timed out while waiting for process to finish", "output:\n---\n%s\n---\n", s.process.CombinedOutput())
	}
}

func (s *Suite) Output() string {
	return s.process.CombinedOutput()
}

func (s *Suite) Expect(value string, timeout ...time.Duration) {
	rx, err := regexp.Compile(regexp.QuoteMeta(value))
	if err != nil {
		s.FailNow("Value is not valid regex", "value: %s", regexp.QuoteMeta(value))
	}
	s.ExpectRe(rx, timeout...)
}

func (s *Suite) ExpectExact(value string, timeout ...time.Duration) {
	rx, err := regexp.Compile("^" + regexp.QuoteMeta(value) + "$")
	if err != nil {
		s.FailNow("Value is not valid regex", "value: %s", regexp.QuoteMeta(value))
	}
	s.ExpectRe(rx, timeout...)
}

func (s *Suite) ExpectRe(value *regexp.Regexp, timeout ...time.Duration) {
	t := 10 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}

	out := ""
	err := s.Timeout(func(stop chan bool) {
		s.process.OnOutput(func(output []byte) {
			if value.MatchString(string(output)) {
				stop <- true
			}
			out = out + string(output)
		})
	}, t)
	if err != nil {
		s.FailNow("Could not meet expectation", "Expectation: '%s'\nError: %v\n---\noutput:\n---\n%s\n---\n",
			value.String(), err, s.process.CombinedOutput())
	}
}

func (s *Suite) Send(value string) {
	// Since we're not running a TTY emulator we need little workarounds like this to ensure stdin is ready
	time.Sleep(100 * time.Millisecond)

	err := s.process.Write(value + "\n")
	if err != nil {
		s.FailNow("Could not send data to stdin", "error: %v", err)
	}
}

func (s *Suite) SendQuit() {
	s.process.Quit()
}

func (s *Suite) Stop() {
	if s.process == nil {
		s.FailNow("stop called without a spawned process")
	}
}

func (s *Suite) Timeout(f func(stop chan bool), t time.Duration) error {
	stop := make(chan bool)
	go func() {
		f(stop)
	}()

	select {
	case <-stop:
		return nil
	case <-s.processEnded:
		return errors.New("Process ended")
	case <-time.After(t):
		return errors.New("Timeout reached")
	}

	panic("I should never be reached")
}
