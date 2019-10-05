// +build external

package main

import (
	"github.com/ActiveState/cli/internal/failures"
)

func runCPUProfiling() (cleanUp func(), fail *failures.Failure) {
	return func() {}, nil
}
