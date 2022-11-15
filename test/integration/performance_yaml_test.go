package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

// Configuration values for the performance tests
const (
	DefaultMaxTime = 100 * time.Millisecond
	DefaultSamples = 10
	// Add other configuration values on per-test basis if needed
)

var (
	rx = regexp.MustCompile(`Profiling: main took .*\((\d+)\)`)
)

type PerformanceYamlIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PerformanceYamlIntegrationTestSuite) startSvc(ts *e2e.Session) {
	// Start svc first, as we don't want to measure svc startup time which would only happen the very first invocation
	stdout, stderr, err := exeutils.ExecSimple(ts.SvcExe, []string{"start"}, []string{})
	suite.Require().NoError(err, fmt.Sprintf("Full error:\n%v\nstdout:\n%s\nstderr:\n%s", errs.JoinMessage(err), stdout, stderr))
}

func (suite *PerformanceYamlIntegrationTestSuite) TestExpandSecret() {
	suite.testScriptPerformance("expand-secret", "WORLD", DefaultSamples, DefaultMaxTime)
}

func (suite *PerformanceYamlIntegrationTestSuite) TestExpandSecretMultiple() {
	suite.testScriptPerformance("expand-secret-multiple", "FOO BAR BAZ", DefaultSamples, DefaultMaxTime)
}

func (suite *PerformanceYamlIntegrationTestSuite) TestEvaluateProjectPath() {
	suite.testScriptPerformance("evaluate-project-path", "", DefaultSamples, DefaultMaxTime)
}

func (suite *PerformanceYamlIntegrationTestSuite) TestUseConstant() {
	suite.testScriptPerformance("use-constant", "foo", DefaultSamples, DefaultMaxTime)
}

func (suite *PerformanceYamlIntegrationTestSuite) TestUseConstantsMultiple() {
	suite.testScriptPerformance("use-constant-multiple", "foo bar baz", DefaultSamples, DefaultMaxTime)
}

func (suite *PerformanceYamlIntegrationTestSuite) TestCallScript() {
	suite.testScriptPerformance("call-script", "Hello World", DefaultSamples, DefaultMaxTime)
}

func (suite *PerformanceYamlIntegrationTestSuite) TestExpandProject() {
	suite.Run("url", func() {
		suite.testScriptPerformance("expand-project-url", "https://platform.activestate.com/ActiveState-CLI/Yaml-Test", DefaultSamples, DefaultMaxTime)
	})
	suite.Run("commit", func() {
		suite.testScriptPerformance("expand-project-commit", "0476ac66-007c-4da7-8922-d6ea9b284fae", DefaultSamples, DefaultMaxTime)
	})
	suite.Run("branch", func() {
		suite.testScriptPerformance("expand-project-branch", "main", DefaultSamples, DefaultMaxTime)
	})
	suite.Run("owner", func() {
		suite.testScriptPerformance("expand-project-owner", "ActiveState-CLI", DefaultSamples, DefaultMaxTime)
	})
	suite.Run("owner", func() {
		suite.testScriptPerformance("expand-project-name", "Yaml-Test", DefaultSamples, DefaultMaxTime)
	})
	suite.Run("namespace", func() {
		suite.testScriptPerformance("expand-project-namespace", "ActiveState-CLI/Yaml-Test", DefaultSamples, DefaultMaxTime)
	})
}

func (suite *PerformanceYamlIntegrationTestSuite) testScriptPerformance(scriptName, expect string, samples int, max time.Duration) {
	suite.OnlyRunForTags(tagsuite.Performance)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.startSvc(ts)

	ts.LoginAsPersistentUser()

	root, err := environment.GetRootPath()
	suite.NoError(err)
	prjPath := filepath.Join(root, "test", "integration", "testdata", "yaml", "activestate.yaml")
	contents, err := fileutils.ReadFile(prjPath)
	suite.NoError(err)

	ts.PrepareActiveStateYAML(string(contents))

	var times []time.Duration
	var total time.Duration
	for x := 0; x < samples; x++ {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("run", scriptName),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_UPDATES=true", "ACTIVESTATE_PROFILE=true"))
		if expect != "" {
			cp.Expect(expect)
		}
		cp.ExpectExitCode(0)
		v := rx.FindStringSubmatch(cp.Snapshot())
		if len(v) < 2 {
			suite.T().Fatalf("Could not find '%s' in output: %s", rx.String(), cp.Snapshot())
		}
		durMS, err := strconv.Atoi(v[1])
		suite.Require().NoError(err)
		dur := time.Millisecond * time.Duration(durMS)
		times = append(times, dur)
		total = total + dur
	}

	avg := total / time.Duration(samples)
	fmt.Println("Average:", avg)

	// TODO: Add check to ensure that the average is within the expected range
}

func TestPerformanceYamlIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceYamlIntegrationTestSuite))
}
