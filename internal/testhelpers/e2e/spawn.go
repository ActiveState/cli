package e2e

import (
	"github.com/ActiveState/termtest"
)

type SpawnOptions func(*termtest.Options) error

func WithArgs(args ...string) SpawnOptions {
	return func(opts *termtest.Options) error {
		opts.Args = args
		return nil
	}
}

func WithWorkDirectory(wd string) SpawnOptions {
	return func(opts *termtest.Options) error {
		opts.WorkDirectory = wd
		return nil
	}
}

func AppendEnv(env ...string) SpawnOptions {
	return func(opts *termtest.Options) error {
		opts.Environment = append(opts.Environment, env...)
		return nil
	}
}

func HideCmdLine() SpawnOptions {
	return func(opts *termtest.Options) error {
		opts.HideCmdLine = true
		return nil
	}
}
