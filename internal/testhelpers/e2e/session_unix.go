//go:build !windows
// +build !windows

package e2e

import "time"

const RuntimeBuildSourcingTimeout = 6 * time.Minute
