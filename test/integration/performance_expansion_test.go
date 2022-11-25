package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
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
	DefaultProject = "https://platform.activestate.com/ActiveState-CLI/Yaml-Test/?branch=main&commitID=0476ac66-007c-4da7-8922-d6ea9b284fae"

	DefaultMaxTime        = 1000 * time.Millisecond
	DefaultSamples        = 10
	DefaultVariance       = 0.75
	DefaultSecretsMaxTime = 2100 * time.Millisecond
	// Add other configuration values on per-test basis if needed
)

type PerformanceExpansionIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PerformanceExpansionIntegrationTestSuite) startSvc(ts *e2e.Session) {
	// Start svc first, as we don't want to measure svc startup time which would only happen the very first invocation
	stdout, stderr, err := exeutils.ExecSimple(ts.SvcExe, []string{"start"}, []string{})
	suite.Require().NoError(err, fmt.Sprintf("Full error:\n%v\nstdout:\n%s\nstderr:\n%s", errs.JoinMessage(err), stdout, stderr))
}

func (suite *PerformanceExpansionIntegrationTestSuite) TestExpansionPerformance() {
	suite.OnlyRunForTags(tagsuite.Performance)
	baseline := DefaultMaxTime
	suite.Run("CallScript", func() {
		avg := suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "call-script",
				Value:    `echo "Hello World"`,
				Language: "bash",
			},
			expect:  "Hello World",
			samples: DefaultSamples,
			max:     DefaultMaxTime,
		})
		variance := float64(avg) + (float64(avg) * DefaultVariance)
		baseline = time.Duration(variance)
	})

	suite.Run("CallScriptFromMerged", func() {
		additionalYamls := make(map[string]projectfile.Project)
		additionalYamls["activestate.test.yaml"] = projectfile.Project{
			Scripts: []projectfile.Script{{Name: "call-script", Value: `echo "Hello World"`}},
		}
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "merged-script",
				Value:    `echo "Hello World"`,
				Language: "bash",
			},
			expect:              "Hello World",
			samples:             DefaultSamples,
			max:                 baseline,
			additionalYamlFiles: additionalYamls,
		})
	})

	suite.Run("EvaluateProjectPath", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "evaluate-project-path",
				Value:    `echo $project.path()`,
				Language: "bash",
			},
			samples: DefaultSamples,
			max:     baseline,
		})
	})

	suite.Run("ExpandProjectBranch", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "expand-project-branch",
				Value:    `echo $project.branch()`,
				Language: "bash",
			},
			expect:  "main",
			samples: DefaultSamples,
			max:     baseline,
		})
	})

	suite.Run("ExpandProjectCommit", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "expand-project-commit",
				Value:    `echo $project.commit()`,
				Language: "bash",
			},
			expect:  "0476ac66-007c-4da7-8922-d6ea9b284fae",
			samples: DefaultSamples,
			max:     baseline,
		})
	})

	suite.Run("ExpandProjectName", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "expand-project-name",
				Value:    `echo $project.name()`,
				Language: "bash",
			},
			expect:  "Yaml-Test",
			samples: DefaultSamples,
			max:     baseline,
		})
	})

	suite.Run("ExpandProjectNamespace", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "expand-project-namespace",
				Value:    `echo $project.namespace()`,
				Language: "bash",
			},
			expect:  "ActiveState-CLI/Yaml-Test",
			samples: DefaultSamples,
			max:     baseline,
		})
	})

	suite.Run("ExpandProjectOwner", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "expand-project-owner",
				Value:    `echo $project.owner()`,
				Language: "bash",
			},
			expect:  "ActiveState-CLI",
			samples: DefaultSamples,
			max:     baseline,
		})
	})

	suite.Run("ExpandProjectURL", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "expand-project-url",
				Value:    `echo $project.url()`,
				Language: "bash",
			},
			expect:  "https://platform.activestate.com/ActiveState-CLI/Yaml-Test",
			samples: DefaultSamples,
			max:     baseline,
		})
	})

	suite.Run("ExpandSecret", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "expand-secret",
				Value:    `echo $secrets.project.HELLO`,
				Language: "bash",
			},
			expect:       "WORLD",
			samples:      DefaultSamples,
			max:          DefaultSecretsMaxTime,
			authRequired: true,
		})
	})

	suite.Run("ExpandSecretMultiple", func() {
		secretsMultipleVariance := float64(DefaultSecretsMaxTime) * 1.25
		secretsMultipleBaseline := time.Duration(secretsMultipleVariance)
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "expand-secret",
				Value:    `echo $secrets.project.FOO $secrets.project.BAR $secrets.project.BAZ`,
				Language: "bash",
			},
			expect:       "FOO BAR BAZ",
			samples:      DefaultSamples,
			max:          secretsMultipleBaseline,
			authRequired: true,
		})
	})

	suite.Run("GetScriptPath", func() {
		expect := ".sh"
		if runtime.GOOS == "windows" {
			expect = ".bat"
		}
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "script-path",
				Value:    `echo $scripts.hello-world.path()`,
				Language: "bash",
			},
			expect:            expect,
			samples:           DefaultSamples,
			max:               baseline,
			additionalScripts: projectfile.Scripts{{Name: "hello-world", Value: `echo "Hello World"`}},
		})
	})

	suite.Run("UseConstant", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "use-constant",
				Value:    `echo $constants.foo`,
				Language: "bash",
			},
			expect:    "foo",
			samples:   DefaultSamples,
			max:       baseline,
			constants: projectfile.Constants{{Name: "foo", Value: "foo"}},
		})
	})

	suite.Run("UseConstantMultiple", func() {
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "use-constant-multiple",
				Value:    `echo $constants.foo $constants.bar $constants.baz`,
				Language: "bash",
			},
			expect:    "foo",
			samples:   DefaultSamples,
			max:       baseline,
			constants: projectfile.Constants{{Name: "foo", Value: "foo"}, {Name: "bar", Value: "bar"}, {Name: "baz", Value: "baz"}},
		})
	})

	suite.Run("UseConstantFromMerged", func() {
		additionalYaml := make(map[string]projectfile.Project)
		additionalYaml["activestate.test.yaml"] = projectfile.Project{
			Constants: projectfile.Constants{{Name: "merged", Value: "merged"}},
		}
		suite.testScriptPerformance(scriptPerformanceOptions{
			script: projectfile.Script{
				Name:     "use-constant-merged",
				Value:    `echo $constants.merged`,
				Language: "bash",
			},
			expect:              "merged",
			samples:             DefaultSamples,
			max:                 baseline,
			additionalYamlFiles: additionalYaml,
		})
	})

}

type scriptPerformanceOptions struct {
	script              projectfile.Script
	expect              string
	samples             int
	max                 time.Duration
	authRequired        bool
	additionalScripts   projectfile.Scripts
	constants           projectfile.Constants
	additionalYamlFiles map[string]projectfile.Project
}

func (suite *PerformanceExpansionIntegrationTestSuite) testScriptPerformance(opts scriptPerformanceOptions) time.Duration {
	suite.OnlyRunForTags(tagsuite.Performance)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.startSvc(ts)

	if opts.authRequired {
		ts.LoginAsPersistentUser()
	}

	projectFile := projectfile.Project{
		Project:   DefaultProject,
		Constants: opts.constants,
		Scripts:   opts.additionalScripts,
	}
	projectFile.Scripts = append(projectFile.Scripts, opts.script)

	contents, err := yaml.Marshal(projectFile)
	suite.NoError(err)

	ts.PrepareActiveStateYAML(string(contents))

	for name, file := range opts.additionalYamlFiles {
		contents, err := yaml.Marshal(file)
		suite.NoError(err)
		suite.prepareAlternateActiveStateYaml(name, string(contents), ts)
	}

	return performanceTest([]string{"run", opts.script.Name}, opts.expect, opts.samples, opts.max, suite.Suite, ts)
}

func (suite *PerformanceExpansionIntegrationTestSuite) prepareAlternateActiveStateYaml(name, contents string, ts *e2e.Session) {
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
	suite.Run(t, new(PerformanceExpansionIntegrationTestSuite))
}
