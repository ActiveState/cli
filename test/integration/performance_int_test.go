package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

// The max time is based on the average execution times across platforms at the time that this was configured
// Increasing this should be a LAST RESORT
var StateVersionMaxTime = 350 * time.Millisecond // DO NOT CHANGE WITHOUT DISCUSSION WITH THE TEAM
var StateVersionTotalSamples = 10

type PerformanceIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PerformanceIntegrationTestSuite) TestShow() {
	suite.OnlyRunForTags(tagsuite.Performance)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Start svc first, as we don't want to measure svc startup time which would only happen the very first invocation
	stdout, stderr, err := exeutils.ExecSimple(ts.SvcExe, []string{"start"}, []string{})
	suite.Require().NoError(err, fmt.Sprintf("Full error:\n%v\nstdout:\n%s\nstderr:\n%s", errs.JoinMessage(err), stdout, stderr))

	var firstEntry string
	times := []time.Duration{}
	var total int64
	for x := 0; x < StateVersionTotalSamples+1; x++ {
		start := time.Now()
		stdout, stderr, err := exeutils.ExecSimple(ts.Exe, []string{"--version"}, []string{"ACTIVESTATE_CLI_DISABLE_UPDATES=true", "ACTIVESTATE_PROFILE=true"})
		suite.Require().NoError(err, fmt.Sprintf("Full error:\n%v\nstdout:\n%s\nstderr:\n%s", errs.JoinMessage(err), stdout, stderr))
		end := time.Since(start)
		if firstEntry == "" {
			firstEntry = stdout
		}
		if x == 0 {
			// Skip the first one as this one will always be slower due to having to wait for state-svc
			continue
		}
		times = append(times, end)
		total += end.Nanoseconds()
	}

	var avg = time.Duration(total / int64(StateVersionTotalSamples))
	if avg.Nanoseconds() > StateVersionMaxTime.Nanoseconds() {
		suite.FailNow(
			fmt.Sprintf(`'state --version' is performing poorly!
Average duration: %s
Minimum: %s
Total: %s
Totals: %v

Output of first run:
%s`,
				avg.String(),
				StateVersionMaxTime.String(),
				time.Duration(total).String(),
				times,
				firstEntry))
	}
}

func TestPerformanceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceIntegrationTestSuite))
}
