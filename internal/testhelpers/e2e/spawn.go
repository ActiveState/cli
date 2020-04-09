package e2e

import (
	"github.com/ActiveState/cli/pkg/expect/conproc"
)

type SpawnOptions func(*conproc.Options) error

func WithArgs(args ...string) SpawnOptions {
	return func(opts *conproc.Options) error {
		opts.Args = args
		return nil
	}
}

func WithWorkDirectory(wd string) SpawnOptions {
	return func(opts *conproc.Options) error {
		opts.WorkDirectory = wd
		return nil
	}
}

func AppendEnv(env ...string) SpawnOptions {
	return func(opts *conproc.Options) error {
		opts.Environment = append(opts.Environment, env...)
		return nil
	}
}
