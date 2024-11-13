package e2e

import "time"

// RuntimeBuildSourcingTimeout uses a higher timeout on Windows cause Windows
const RuntimeBuildSourcingTimeout = 12 * time.Minute
