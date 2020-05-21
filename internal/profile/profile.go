package profile

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
)

var (
	// FailSetupCPUProfiling indicates a failure setting up cpu profiling.
	FailSetupCPUProfiling = failures.Type("profile.fail.setup.cpu", failures.FailNonFatal)
)

// CPU runs the CPU profiler. Be sure to run the cleanup func.
func CPU() (cleanUp func(), err error) {
	timeString := time.Now().Format("20060102-150405.000")
	timeString = strings.Replace(timeString, ".", "-", 1)
	cpuProfFile := fmt.Sprintf("cpu_%s.prof", timeString)

	f, err := os.Create(cpuProfFile)
	if err != nil {
		return func() {}, errs.Wrap(err, "Could not create CPU profiling file: %s", cpuProfFile)
	}

	if err = pprof.StartCPUProfile(f); err != nil {
		return func() {}, errs.Wrap(err, "Could not start CPU profiling")
	}

	return pprof.StopCPUProfile, nil
}
