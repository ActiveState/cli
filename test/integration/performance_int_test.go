package integration

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

// The max time is based on the median execution times across platforms at the time that this was configured
// Increasing this should be a LAST RESORT
var StateVersionMaxTime = 100 * time.Millisecond // DO NOT CHANGE WITHOUT DISCUSSION WITH THE TEAM
var StateVersionTotalSamples = 10

type PerformanceIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PerformanceIntegrationTestSuite) TestVersionPerformance() {
	suite.OnlyRunForTags(tagsuite.Performance)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Start svc first, as we don't want to measure svc startup time which would only happen the very first invocation
	stdout, stderr, err := osutils.ExecSimple(ts.SvcExe, []string{"start"}, []string{})
	suite.Require().NoError(err, fmt.Sprintf("Full error:\n%v\nstdout:\n%s\nstderr:\n%s", errs.JoinMessage(err), stdout, stderr))

	performanceTest([]string{"--version"}, "", StateVersionTotalSamples, StateVersionMaxTime, false, &suite.Suite, ts)
}

func TestPerformanceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceIntegrationTestSuite))
}

func performanceTest(commands []string, expect string, samples int, maxTime time.Duration, verbose bool, suite *tagsuite.Suite, ts *e2e.Session) time.Duration {
	rx := regexp.MustCompile(`Profiling: main took .*\((\d+)\)`)
	var firstEntry, firstLogs string
	times := []time.Duration{}
	var total time.Duration
	for x := 0; x < samples+1; x++ {
		opts := []e2e.SpawnOptSetter{
			e2e.OptArgs(commands...),
			e2e.OptAppendEnv(
				constants.DisableUpdates+"=true",
				constants.ProfileEnvVarName+"=true",
				constants.DisableRuntime+"=true",
			),
		}
		termtestLogs := &bytes.Buffer{}
		if verbose {
			opts = append(opts, e2e.OptTermTest(func(o *termtest.Opts) error {
				o.Logger = log.New(termtestLogs, "TermTest: ", log.LstdFlags|log.Lshortfile)
				return nil
			}))
		}
		cp := ts.SpawnWithOpts(opts...)
		if expect != "" {
			cp.Expect(expect)
		}
		cp.ExpectExitCode(0)
		logs, err := io.ReadAll(termtestLogs)
		suite.NoError(err)
		v := rx.FindStringSubmatch(cp.Output())
		if len(v) < 2 {
			suite.T().Fatalf("Could not find '%s' in output:\n%s\n\ntermtest logs:\n%s", rx.String(), cp.Output(), logs)
		}
		durMS, err := strconv.Atoi(v[1])
		suite.Require().NoError(err)
		dur := time.Millisecond * time.Duration(durMS)

		if firstEntry == "" {
			firstEntry = cp.Output()
			firstLogs = ts.DebugLogsDump()
		}
		if x == 0 {
			// Skip the first one as this one will always be slower due to having to wait for state-svc or sourcing a runtime
			// Also pause for a second allow the second run to use the cached results from the first
			time.Sleep(1 * time.Second)
			continue
		}
		times = append(times, dur)
		total = total + dur
	}

	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	mid := len(times) / 2
	var median time.Duration
	if len(times)%2 == 0 {
		median = (times[mid-1] + times[mid]) / 2
	} else {
		median = times[mid]
	}

	if median.Milliseconds() > maxTime.Milliseconds() {
		suite.FailNow(
			fmt.Sprintf(`'%s' is performing poorly!
	Median duration: %s
	Maximum: %s
	Total: %s
	Totals: %v

	Output of first run:
	%s

	%s`,
				strings.Join(commands, " "),
				median.String(),
				maxTime.String(),
				time.Duration(total).String(),
				times,
				firstEntry,
				firstLogs))
	}

	return median
}
