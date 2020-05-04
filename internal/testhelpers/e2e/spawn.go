package e2e

import (
	"strings"

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
		opts.Environment = appendEnv(opts.Environment, env...)
		return nil
	}
}


func appendEnv(currentEnv []string, env ...string) []string {		// Scan for duplicates
	for _, v := range env {
		k := strings.Split(v, "=")[0]
		for i, vv := range currentEnv {
			if strings.HasPrefix(vv, k+"=") {
				// Delete duplicate
				ev := currentEnv
				currentEnv = append(ev[:i], ev[i+1:]...)
			}
		}
	}

	currentEnv = append(currentEnv, env...)
	return currentEnv
}
