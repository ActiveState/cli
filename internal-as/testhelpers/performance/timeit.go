package performance

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/errs"
)

// TimeIt tests if the average duration of the function is less than the maxDuration
func TimeIt(t *testing.T, f func(), attempts int, maxDuration time.Duration) error {
	durations := make([]time.Duration, attempts)
	avgDuration := time.Duration(0)
	for i := 0; i < attempts; i++ {
		start := time.Now()
		f()
		durations[i] = time.Since(start)
		avgDuration += durations[i]
	}

	avgDuration /= time.Duration(attempts)

	if avgDuration > maxDuration {
		return errs.New("Average duration of %s exceeded max duration of %s, individual durations: %#v", avgDuration, maxDuration, durations)
	}

	return nil
}
