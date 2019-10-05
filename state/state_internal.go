// +build !external

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/state/internal/profile"
)

func runCPUProfiling() (cleanUp func(), fail *failures.Failure) {
	timeString := time.Now().Format("20060102-150405.000")
	timeString = strings.Replace(timeString, ".", "-", 1)
	cpuProfFile := fmt.Sprintf("cpu_%s.prof", timeString)

	cleanUpCPU, fail := profile.CPU(cpuProfFile)
	if fail != nil {
		return nil, fail
	}

	logging.Debug(fmt.Sprintf("profiling cpu (%s)", cpuProfFile))

	cleanUp = func() {
		logging.Debug("cleaning up cpu profiling")
		cleanUpCPU()
	}

	return cleanUp, nil
}
