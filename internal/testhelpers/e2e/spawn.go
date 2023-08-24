package e2e

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/termtest"
)

type SpawnedCmd struct {
	*termtest.TermTest
	opts SpawnOpts
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
	rx, err := regexp.Compile(v)
	if err != nil {
		return errs.Wrap(err, "ExpectRe received invalid regex string")
	}
	return s.TermTest.ExpectRe(rx, opts...)
}

func (s *SpawnedCmd) ExpectInput(opts ...termtest.SetExpectOpt) error {
	cmdName := strings.TrimSuffix(strings.ToLower(filepath.Base(s.Cmd().Path)), ".exe")
	if !sliceutils.Contains([]string{"bash", "zsh", "cmd"}, cmdName) {
		return errs.New("ExpectInput can only be used with bash, zsh, or cmd")
	}

	send := `echo $'expect\'input'`
	expect := `expect'input`
	if cmdName == "cmd" {
		send = `echo ^<expect input^>`
		expect = `<expect input>`
	}

	if err := s.SendLine(send); err != nil {
		return fmt.Errorf("could not send line to terminal: %w", err)
	}

	return s.Expect(expect, opts...)
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
