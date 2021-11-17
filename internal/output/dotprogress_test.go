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
	interval := 10 * time.Millisecond
	noIntervals := 10
	sleepTime := time.Duration(noIntervals+1) * interval
	dp := output.NewDotProgress(out, "Progress", interval)
	time.Sleep(sleepTime)
	dp.Stop("Done")
	dots := strings.Repeat(".", noIntervals)
	require.Regexp(t, regexp.MustCompile("Progress..."+dots+"\\.* Done"), out.ErrorOutput())
}
