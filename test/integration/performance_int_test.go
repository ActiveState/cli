package integration

import (
	"fmt"
	"regexp"
	"strconv"
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
var StateVersionMaxTime = 30 * time.Millisecond // DO NOT CHANGE WITHOUT DISCUSSION WITH THE TEAM
var StateVersionTotalSamples = 10

type PerformanceIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PerformanceIntegrationTestSuite) TestVersionPerformance() {
	suite.OnlyRunForTags(tagsuite.Performance)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Start svc first, as we don't want to measure svc startup time which would only happen the very first invocation
	stdout, stderr, err := exeutils.ExecSimple(ts.SvcExe, []string{"start"}, []string{})
	suite.Require().NoError(err, fmt.Sprintf("Full error:\n%v\nstdout:\n%s\nstderr:\n%s", errs.JoinMessage(err), stdout, stderr))

	rx := regexp.MustCompile(`Profiling: main took .*\((\d+)\)`)

	var firstEntry, firstStateLog, firstSvcLog string
	times := []time.Duration{}
	var total time.Duration
	for x := 0; x < StateVersionTotalSamples+1; x++ {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("--version"),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_UPDATES=true", "ACTIVESTATE_PROFILE=true"))
		cp.ExpectExitCode(0)
		v := rx.FindStringSubmatch(cp.Snapshot())
		if len(v) < 2 {
			suite.T().Fatalf("Could not find '%s' in output: %s", rx.String(), cp.Snapshot())
		}
		durMS, err := strconv.Atoi(v[1])
		suite.Require().NoError(err)
		dur := time.Millisecond * time.Duration(durMS)

		if firstEntry == "" {
			firstEntry = cp.Snapshot()
			firstStateLog = ts.MostRecentStateLog()
			firstSvcLog = ts.SvcLog()
		}
		if x == 0 {
			// Skip the first one as this one will always be slower due to having to wait for state-svc
			continue
		}
		times = append(times, dur)
		total = total + dur
	}

	var avg = total / time.Duration(StateVersionTotalSamples)
	if avg.Milliseconds() > StateVersionMaxTime.Milliseconds() {
		suite.FailNow(
			fmt.Sprintf(`'state --version' is performing poorly!
Average duration: %s
Minimum: %s
Total: %s
Totals: %v

Output of first run:
%s

State Tool log:
%s

Svc log:
%s`,
				avg.String(),
				StateVersionMaxTime.String(),
				time.Duration(total).String(),
				times,
				firstEntry,
				firstStateLog,
				firstSvcLog))
	}
}

func TestPerformanceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceIntegrationTestSuite))
}
