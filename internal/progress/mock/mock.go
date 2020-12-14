package mock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/progress"
)

var _ progress.Incrementer = &Incrementer{}

// TestProgress is wrapper around a Progress that can be used in test to ensure that the progress bar
// loop terminates after a configurable time-out.
// This construct has been added, due to numerous problems with hanging progress bar loops.
type TestProgress struct {
	*progress.Progress
}

// NewTestProgress returns a new testable progress-bar for unit tests
func NewTestProgress() *TestProgress {
	p := progress.New(progress.WithOutput(nil))
	return &TestProgress{
		Progress: p,
	}
}

// Close should be run at the end of the test.  It ensures that all resources are released
func (tp *TestProgress) Close() {
	tp.Cancel()
	tp.Progress.Close()
}

// AssertProperClose asserts that the progress bar loop terminated normally (with all sub-bars completed at 100%)
func (tp *TestProgress) AssertProperClose(t *testing.T) {
	tp.assertClose(t, time.Second*2, true)
}

// AssertCloseAfterCancellation asserts that the progress bar loop terminated due to a cancellation event
func (tp *TestProgress) AssertCloseAfterCancellation(t *testing.T) {
	tp.assertClose(t, time.Second*2, false)
}

func (tp *TestProgress) assertClose(t *testing.T, after time.Duration, properShutdown bool) {
	done := make(chan struct{})
	go func() {
		tp.Progress.Close()
		close(done)
	}()
	select {
	case <-done:
		if properShutdown {
			assert.False(t, tp.HasBeenCancelled(), "progress bar shut down without cancellation")
		} else {
			assert.True(t, tp.HasBeenCancelled(), "progress bar shut down after cancellation")
		}
	case <-time.After(after):
		tp.Cancel()
		t.Error("Timed out waiting for progress bar to shut down.  Either a bar did not complete, or you forgot to call the Cancel() method.")
		<-done
	}
}

// Incrementer implements a simple counter.  This can be used to test functions that are expected to report its progress incrementally
type Incrementer struct {
	Count int
}

func NewMockIncrementer() *Incrementer {
	return &Incrementer{Count: 0}
}

// Increment increments the progress count by one
func (mi *Incrementer) Increment(_ ...time.Duration) {
	mi.Count++
}
