package profile

import (
	"os"
	"runtime/pprof"

	"github.com/ActiveState/cli/internal/failures"
)

var (
	// FailSetupCPUProfiling indicates a failure setting up cpu profiling.
	FailSetupCPUProfiling = failures.Type("profile.fail.setup.cpu", failures.FailNonFatal)
)

// CPU runs the CPU profiler. Be sure to run the cleanup func.
func CPU(file string) (cleanUp func(), fail *failures.Failure) {
	if file == "" {
		return func() {}, nil
	}

	f, err := os.Create(file)
	if err != nil {
		return func() {}, FailSetupCPUProfiling.Wrap(err)
	}
	if err = pprof.StartCPUProfile(f); err != nil {
		return func() {}, FailSetupCPUProfiling.Wrap(err)
	}

	fn := func() {
		pprof.StopCPUProfile()
	}
	return fn, nil
}
