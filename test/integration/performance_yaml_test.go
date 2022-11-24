package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

// Configuration values for the performance tests
const (
	DefaultMaxTime  = 1000 * time.Millisecond
	DefaultSamples  = 10
	DefaultVariance = 0.2
	SecretsVariance = 2.4
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

func (suite *PerformanceYamlIntegrationTestSuite) TestYamlPerformance() {
	suite.OnlyRunForTags(tagsuite.Performance)
	baseline := DefaultMaxTime
	suite.Run("CallScript", func() {
		avg := suite.testScriptPerformance("call-script", "Hello World", DefaultSamples, DefaultMaxTime)
		variance := float64(avg) + (float64(avg) * DefaultVariance)
		baseline = time.Duration(variance)
	})

	suite.Run("CallScriptFromMerged", func() {
		suite.testScriptPerformance("merged-script", "Hello World", DefaultSamples, baseline)
	})

	suite.Run("EvaluateProjectPath", func() {
		suite.testScriptPerformance("evaluate-project-path", "", DefaultSamples, baseline)
	})

	suite.Run("ExpandProjectBranch", func() {
		suite.testScriptPerformance("expand-project-branch", "main", DefaultSamples, baseline)
	})

	suite.Run("ExpandProjectCommit", func() {
		suite.testScriptPerformance("expand-project-commit", "0476ac66-007c-4da7-8922-d6ea9b284fae", DefaultSamples, baseline)
	})

	suite.Run("ExpandProjectName", func() {
		suite.testScriptPerformance("expand-project-name", "Yaml-Test", DefaultSamples, baseline)
	})

	suite.Run("ExpandProjectNamespace", func() {
		suite.testScriptPerformance("expand-project-namespace", "ActiveState-CLI/Yaml-Test", DefaultSamples, baseline)
	})

	suite.Run("ExpandProjectOwner", func() {
		suite.testScriptPerformance("expand-project-owner", "ActiveState-CLI", DefaultSamples, baseline)
	})

	suite.Run("ExpandProjectURL", func() {
		suite.testScriptPerformance("expand-project-url", "https://platform.activestate.com/ActiveState-CLI/Yaml-Test", DefaultSamples, baseline)
	})

	suite.Run("ExpandSecret", func() {
		secretsVariance := float64(baseline) * SecretsVariance
		secretsBaseline := time.Duration(secretsVariance)
		suite.testScriptPerformance("expand-secret", "WORLD", DefaultSamples, secretsBaseline)
	})

	suite.Run("ExpandSecretMultiple", func() {
		secretsMultipleVariance := float64(baseline) * (1.25 * SecretsVariance)
		secretsMultipleBaseline := time.Duration(secretsMultipleVariance)
		suite.testScriptPerformance("expand-secret-multiple", "FOO BAR BAZ", DefaultSamples, secretsMultipleBaseline)
	})

	suite.Run("GetScriptPath", func() {
		suite.testScriptPerformance("script-path", ".sh", DefaultSamples, baseline)
	})

	suite.Run("UseConstant", func() {
		suite.testScriptPerformance("use-constant", "foo", DefaultSamples, baseline)
	})

	suite.Run("UseConstantMultiple", func() {
		suite.testScriptPerformance("use-constant-multiple", "foo bar baz", DefaultSamples, baseline)
	})

	suite.Run("UseConstantFromMerged", func() {
		suite.testScriptPerformance("use-constant-multiple", "foo bar baz", DefaultSamples, baseline)
	})

}

func (suite *PerformanceYamlIntegrationTestSuite) testScriptPerformance(scriptName, expect string, samples int, max time.Duration) time.Duration {
	suite.OnlyRunForTags(tagsuite.Performance)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.startSvc(ts)

	ts.LoginAsPersistentUser()

	root, err := environment.GetRootPath()
	suite.NoError(err)
	prjPath := filepath.Join(root, "test", "integration", "testdata", "yaml", "activestate.yaml")
	contents, err := fileutils.ReadFile(prjPath)
	suite.NoError(err)

	ts.PrepareActiveStateYAML(string(contents))

	alternateFileName := "activestate.test.yaml"
	alternatePrjPath := filepath.Join(root, "test", "integration", "testdata", "yaml", alternateFileName)
	contents, err = fileutils.ReadFile(alternatePrjPath)
	suite.NoError(err)

	suite.prepareAlternateActiveStateYaml(alternateFileName, string(contents), ts)

	return performanceTest([]string{"run", scriptName}, expect, samples, max, suite.Suite, ts)
}

func (suite *PerformanceYamlIntegrationTestSuite) prepareAlternateActiveStateYaml(name, contents string, ts *e2e.Session) {
	msg := "cannot setup activestate.yaml file"

	contents = strings.TrimSpace(contents)
	projectFile := &projectfile.Project{}

	err := yaml.Unmarshal([]byte(contents), projectFile)
	suite.NoError(err, msg)

	cfg, err := config.New()
	suite.NoError(err)
	defer func() { suite.NoError(cfg.Close()) }()

	path := filepath.Join(ts.Dirs.Work, name)
	err = fileutils.WriteFile(path, []byte(contents))
	suite.NoError(err, msg)
	suite.True(fileutils.FileExists(path))
}

func TestPerformanceYamlIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceYamlIntegrationTestSuite))
}
