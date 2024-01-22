//go:build !windows
// +build !windows

package e2e

import (
	"time"

	"github.com/ActiveState/termtest"
)

var (
	RuntimeSourcingTimeoutOpt = termtest.OptExpectTimeout(3 * time.Minute)
)
