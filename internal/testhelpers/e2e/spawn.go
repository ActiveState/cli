package e2e

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
)

type SpawnOptions func(*Options) error

func WithArgs(args ...string) SpawnOptions {
	return func(opts *Options) error {
		opts.Options.Args = args
		return nil
	}
}

func WithWorkDirectory(wd string) SpawnOptions {
	return func(opts *Options) error {
		opts.Options.WorkDirectory = wd
		return nil
	}
}

func AppendEnv(env ...string) SpawnOptions {
	return func(opts *Options) error {
		opts.Options.Environment = append(opts.Options.Environment, env...)
		return nil
	}
}

func HideCmdLine() SpawnOptions {
	return func(opts *Options) error {
		opts.Options.HideCmdLine = true
		return nil
	}
}

// NonWriteableBinDir removes the write permission from the directory where the executables are run from.
// This can be used to simulate an installation in a global installation directory that requires super-user rights.
func NonWriteableBinDir() SpawnOptions {
	return func(opts *Options) error {
		opts.NonWriteableBinDir = true
		return nil
	}
}

func BackgroundProcess() SpawnOptions {
	return func(opts *Options) error {
		opts.BackgroundProcess = true
		return nil
	}
}

type Shell string

const (
	Bash Shell = "bash"
	Zsh        = "zsh"
	Tcsh       = "tcsh"
	Fish       = "fish"
	Cmd        = "cmd.exe"
)

func WithShell(shell Shell, s *Session) SpawnOptions {
	return func(opts *Options) error {
		if len(opts.Options.Args) == 0 {
			return errs.New("e2e.WithShell must come after e2e.WithArgs")
		}
		opts.Options.CmdName = string(shell)
		shellArg := "-c"
		if shell == Cmd {
			shellArg = "/k"
		}
		cmd := s.Exe + " " + strings.Join(opts.Options.Args, " ")
		opts.Options.Args = []string{shellArg, cmd} // e.g. -c "state activate project/org"
		return nil
	}
}
