package e2e

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

// ReUseExecutable skips the step of copying the executable to a session directory and instead uses the executable that is in that directory already
func ReUseExecutable() SpawnOptions {
	return func(opts *Options) error {
		opts.ReUseExecutables = true
		return nil
	}
}
