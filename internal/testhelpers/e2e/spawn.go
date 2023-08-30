package e2e

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
)

type SpawnedCmd struct {
	*termtest.TermTest
	opts SpawnOpts
}

func (s *SpawnedCmd) Output() string {
	if runtime.GOOS != "windows" {
		return s.TermTest.Output()
	}
	// For some reason on Windows the PTY starts with the path of the executable followed by the unicode Alert character (U+0007)
	// For now we are working around this bug here
	out := s.TermTest.Output()
	if i := strings.Index(out, "\a"); i != -1 {
		out = out[i+1:]
	}
	return out
}

func (s *SpawnedCmd) WorkDirectory() string {
	return s.TermTest.Cmd().Dir
}

func (s *SpawnedCmd) Wait() error {
	return s.TermTest.Wait(30 * time.Second)
}

func (s *SpawnedCmd) Executable() string {
	return s.TermTest.Cmd().Path
}

func (s *SpawnedCmd) ExpectRe(v string, opts ...termtest.SetExpectOpt) error {
	expectOpts, err := termtest.NewExpectOpts(opts...)
	if err != nil {
		err = fmt.Errorf("could not create expect options: %w", err)
		return s.ExpectErrorHandler(&err, expectOpts)
	}

	rx, err := regexp.Compile(v)
	if err != nil {
		err = errs.Wrap(err, "ExpectRe received invalid regex string")
		return s.ExpectErrorHandler(&err, expectOpts)
	}
	return s.TermTest.ExpectRe(rx, opts...)
}

func (s *SpawnedCmd) ExpectInput(opts ...termtest.SetExpectOpt) error {
	expectOpts, err := termtest.NewExpectOpts(opts...)
	if err != nil {
		err = fmt.Errorf("could not create expect options: %w", err)
		return s.ExpectErrorHandler(&err, expectOpts)
	}

	cmdName := strings.TrimSuffix(strings.ToLower(filepath.Base(s.Cmd().Path)), ".exe")

	shellName := ""
	envMap := osutils.EnvSliceToMap(s.Cmd().Env)
	if v, ok := envMap["SHELL"]; ok {
		shellName = strings.TrimSuffix(strings.ToLower(filepath.Base(v)), ".exe")
	}

	send := `echo $'expect\'input from posix shell'`
	expect := `expect'input from posix shell`
	if cmdName != "bash" && shellName != "bash" && runtime.GOOS == "windows" {
		send = `echo ^<expect input from cmd prompt^>`
		expect = `<expect input from cmd prompt>`
	}

	// Termtest internal functions already implement error handling
	if err := s.SendLine(send); err != nil {
		return fmt.Errorf("could not send line to terminal: %w", err)
	}

	return s.Expect(expect, opts...)
}

func (s *SpawnedCmd) SendEnter() error {
	return s.SendLine("")
}

func (s *SpawnedCmd) SendKeyUp() error {
	return s.Send(string([]byte{0033, '[', 'A'})) // move cursor down
}

func (s *SpawnedCmd) SendKeyDown() error {
	return s.Send(string([]byte{0033, '[', 'B'})) // move cursor down
}

func (s *SpawnedCmd) SendKeyRight() error {
	return s.Send(string([]byte{0033, '[', 'C'})) // move cursor down
}

func (s *SpawnedCmd) SendKeyLeft() error {
	return s.Send(string([]byte{0033, '[', 'D'})) // move cursor down
}

type SpawnOpts struct {
	Args           []string
	Env            []string
	Dir            string
	TermtestOpts   []termtest.SetOpt
	HideCmdArgs    bool
	RunInsideShell bool
}

func NewSpawnOpts() SpawnOpts {
	return SpawnOpts{
		RunInsideShell: false,
	}
}

type SpawnOptSetter func(opts *SpawnOpts)

func OptArgs(args ...string) SpawnOptSetter {
	return func(opts *SpawnOpts) {
		opts.Args = args
	}
}

func OptWD(wd string) SpawnOptSetter {
	return func(opts *SpawnOpts) {
		opts.Dir = wd
	}
}

func OptAppendEnv(env ...string) SpawnOptSetter {
	return func(opts *SpawnOpts) {
		opts.Env = append(opts.Env, env...)
	}
}

func OptTermTest(opt ...termtest.SetOpt) SpawnOptSetter {
	return func(opts *SpawnOpts) {
		opts.TermtestOpts = append(opts.TermtestOpts, opt...)
	}
}

func OptHideArgs() SpawnOptSetter {
	return func(opts *SpawnOpts) {
		opts.HideCmdArgs = true
	}
}

func OptRunInsideShell(v bool) SpawnOptSetter {
	return func(opts *SpawnOpts) {
		opts.RunInsideShell = v
	}
}
