package output_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/stretchr/testify/require"
)

func Test_Dotprogress(t *testing.T) {
	out := outputhelper.NewCatcher()
	interval := 1 * time.Millisecond
	noIntervals := 100
	sleepTime := time.Duration(noIntervals+1) * interval
	dp := output.StartSpinner(out, "Progress", interval)
	time.Sleep(sleepTime)
	dp.Stop("Done")
	dots := strings.Repeat(".", (noIntervals / 20)) // To avoid race conditions we're only counting half the supposed dots

	// GitHub CI's shell sometimes glitches and mixes up stdout, mostly on Windows, but
	// occasionally on other platforms. Allow for the final space before "Done" to occur anywhere
	// before it.
	require.Regexp(t, regexp.MustCompile("Progress..."+dots+"(\\.* |\\.* \\.+)Done"), out.ErrorOutput())
}
