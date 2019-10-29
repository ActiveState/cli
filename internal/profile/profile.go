package profile

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/failures"
)

var (
	// FailSetupCPUProfiling indicates a failure setting up cpu profiling.
	FailSetupCPUProfiling = failures.Type("profile.fail.setup.cpu", failures.FailNonFatal)
)

// CPU runs the CPU profiler. Be sure to run the cleanup func.
func CPU() (cleanUp func(), fail *failures.Failure) {
	timeString := time.Now().Format("20060102-150405.000")
	timeString = strings.Replace(timeString, ".", "-", 1)
	cpuProfFile := fmt.Sprintf("cpu_%s.prof", timeString)

	f, err := os.Create(cpuProfFile)
	if err != nil {
		return func() {}, FailSetupCPUProfiling.Wrap(err)
	}

	if err = pprof.StartCPUProfile(f); err != nil {
		return func() {}, FailSetupCPUProfiling.Wrap(err)
	}

	return pprof.StopCPUProfile, nil
}
