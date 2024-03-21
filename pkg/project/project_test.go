package project_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type ProjectTestSuite struct {
	suite.Suite
	projectFile *projectfile.Project
	project     *project.Project
	testdataDir string
}

func (suite *ProjectTestSuite) BeforeTest(suiteName, testName string) {
	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")

	suite.testdataDir = filepath.Join(root, "pkg", "project", "testdata")
	err = os.Chdir(suite.testdataDir)
	suite.Require().NoError(err, "Should change dir without issue.")
	projectFile, err := projectfile.FromEnv()
	suite.Require().NoError(err, errs.JoinMessage(err))
	suite.projectFile = projectFile
	suite.Require().Nil(err, "Should retrieve projectfile without issue.")
	suite.project, err = project.FromWD()
	suite.Require().Nil(err, "Should retrieve project without issue.")

	cfg, err := config.New()
	suite.Require().NoError(err)
	project.RegisterConditional(constraints.NewPrimeConditional(nil, suite.project, subshell.New(cfg).Shell()))
}

func (suite *ProjectTestSuite) TestGet() {
	config, err := project.FromWD()
	suite.NoError(err, "Run without failure")
	suite.NotNil(config, "Config should be set")
}

func (suite *ProjectTestSuite) TestGetSafe() {
	val, err := project.FromWD()
	suite.NoError(err, "Run without failure")
	suite.NotNil(val, "Config should be set")
}

func (suite *ProjectTestSuite) TestProject() {
	projectLine := "https://platform.activestate.com/ActiveState/project?branch=main&commitID=00010001-0001-0001-0001-000100010001"
	suite.Equal(projectLine, suite.project.URL(), "Values should match")
	suite.Equal("project", suite.project.Name(), "Values should match")
	commitID := suite.project.LegacyCommitID() // Not using localcommit due to import cycle. See anti-pattern comment in localcommit pkg.
	suite.Equal("00010001-0001-0001-0001-000100010001", commitID, "Values should match")
	suite.Equal("ActiveState", suite.project.Owner(), "Values should match")
	suite.Equal("ActiveState/project", suite.project.Namespace().String(), "Values should match")
	suite.Equal("something", suite.project.Environments(), "Values should match")
	suite.Equal("1.0.0-SHA123", suite.project.Version(), "Values should match")
}

func (suite *ProjectTestSuite) TestWhenInSubDirectories() {
	err := os.Chdir(filepath.Join(suite.testdataDir, "sub1", "sub2"))
	suite.Require().NoError(err, "Should change dir without issue.")

	suite.Equal("project", suite.project.Name(), "Values should match")
	suite.Equal("ActiveState", suite.project.Owner(), "Values should match")
	suite.Equal("ActiveState/project", suite.project.Namespace().String(), "Values should match")
	suite.Equal("something", suite.project.Environments(), "Values should match")
	suite.Equal("1.0.0-SHA123", suite.project.Version(), "Values should match")
}

func (suite *ProjectTestSuite) TestEvents() {
	events := suite.project.Events()
	var name string
	switch runtime.GOOS {
	case "linux":
		name = "foo"
	case "windows":
		name = "bar"
	case "darwin":
		name = "baz"
	}

	event := events[0]
	value, err := event.Value()
	suite.NoError(err)

	suite.Equal(name, event.Name(), "Names should match")
	suite.Equal(name+" project", value, "Value should match")
}

func (suite *ProjectTestSuite) TestEventByName() {
	var name string
	switch runtime.GOOS {
	case "linux":
		name = "foo"
	case "windows":
		name = "bar"
	case "darwin":
		name = "baz"
	}

	event := suite.project.EventByName(name, false)
	suite.Equal(name, event.Name())

	event = suite.project.EventByName("not-there", false)
	suite.Nil(event)
}

func (suite *ProjectTestSuite) TestScripts() {
	scripts := suite.project.Scripts()
	var name string
	switch runtime.GOOS {
	case "linux":
		name = "foo"
	case "windows":
		name = "bar"
	case "darwin":
		name = "baz"
	}

	script := scripts[0]
	value, err := script.Value()
	suite.NoError(err)
	raw := script.Raw()
	safe := script.LanguageSafe()
	standalone := script.Standalone()

	suite.Equal(name, script.Name(), "Names should match")
	suite.Equal(name+" project", value, "Value should match")
	suite.Equal(name+" $project.name()", raw, "Raw value should match")
	if runtime.GOOS == "windows" {
		suite.Equal([]language.Language{language.Batch}, safe, "Safe language should match")
	} else {
		suite.Equal([]language.Language{language.Sh}, safe, "Safe language should match")
	}
	suite.True(standalone, "Standalone value should match")
}

func (suite *ProjectTestSuite) TestScriptByName() {
	script := suite.project.ScriptByName("noop")
	suite.Nil(script)

	if runtime.GOOS == "linux" {
		script = suite.project.ScriptByName("foo")
		suite.Require().NotNil(script)
		suite.Equal("foo", script.Name(), "Names should match (Linux)")
		v, err := script.Value()
		suite.NoError(err)
		suite.Equal("foo project", v, "Value should match (Linux)")
		suite.True(script.Standalone(), "Standalone value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		script = suite.project.ScriptByName("bar")
		suite.Require().NotNil(script)
		suite.Equal("bar", script.Name(), "Name should match (Windows)")
		v, err := script.Value()
		suite.NoError(err)
		suite.Equal("bar project", v, "Value should match (Windows)")
		suite.True(script.Standalone(), "Standalone value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		script = suite.project.ScriptByName("baz")
		suite.Require().NotNil(script)
		suite.Equal("baz", script.Name(), "Names should match (OSX)")
		v, err := script.Value()
		suite.NoError(err)
		suite.Equal("baz project", v, "Value should match (OSX)")
		suite.True(script.Standalone(), "Standalone value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestConstants() {
	constants := suite.project.Constants()
	var name string
	switch runtime.GOOS {
	case "linux":
		name = "foo"
	case "windows":
		name = "bar"
	case "darwin":
		name = "baz"
	}

	constant := constants[0]

	value, err := constant.Value()
	suite.NoError(err)

	suite.Equal(name, constant.Name(), "Names should match")
	suite.Equal(name+" project", value, "Value should match")
}

func (suite *ProjectTestSuite) TestSecrets() {
	prj, err := project.FromWD()
	suite.NoError(err, "Run without failure")
	cfg, err := config.New()
	suite.Require().NoError(err)
	defer func() { suite.Require().NoError(cfg.Close()) }()
	auth := authentication.New(cfg)
	secrets := prj.Secrets(cfg, auth)
	suite.Len(secrets, 2)

	userSecret := prj.SecretByName("secret", project.SecretScopeUser, cfg, auth)
	suite.Require().NotNil(userSecret)
	suite.Equal("secret-user", userSecret.Description())
	suite.True(userSecret.IsUser())
	suite.False(userSecret.IsProject())
	suite.Equal("user", userSecret.Scope())

	projectSecret := prj.SecretByName("secret", project.SecretScopeProject, cfg, auth)
	suite.Require().NotNil(projectSecret)
	suite.Equal("secret-project", projectSecret.Description())
	suite.True(projectSecret.IsProject())
	suite.False(projectSecret.IsUser())
	suite.Equal("project", projectSecret.Scope())

	// Value and Save not tested here as they require refactoring so we can test against interfaces (out of scope at this time)
	// https://www.pivotaltracker.com/story/show/166586988
}

func Test_ProjectTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectTestSuite))
}
