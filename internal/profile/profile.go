package profile

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/felixge/fgprof"

	"github.com/ActiveState/cli/internal/errs"
)

var profilingEnabled = false

func init() {
	profilingEnabled = os.Getenv(constants.ProfileEnvVarName) == "true"
}

// CPU runs the CPU profiler. Be sure to run the cleanup func.
func CPU() (cleanUp func() error, err error) {
	timeString := time.Now().Format("20060102-150405.000")
	timeString = strings.Replace(timeString, ".", "-", 1)
	cpuProfFile := fmt.Sprintf("cpu_%s.prof", timeString)

	f, err := os.Create(cpuProfFile)
	if err != nil {
		return func() error { return nil }, errs.Wrap(err, "Could not create CPU profiling file: %s", cpuProfFile)
	}

	return fgprof.Start(f, fgprof.FormatPprof), nil
}

func Measure(name string, start time.Time) {
	if profilingEnabled {
		msg := fmt.Sprintf("Profiling: %s took %s (%d)\n", name, time.Since(start), time.Since(start).Milliseconds())
		logging.Debug(msg)
		fmt.Print(msg)
	}
}