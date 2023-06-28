package e2e

import "github.com/ActiveState/termtest"

type SpawnOpts struct {
	Args         []string
	Env          []string
	Dir          string
	TermtestOpts []termtest.SetOpt
	HideCmdArgs  bool
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
