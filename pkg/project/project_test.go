package project_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/pkg/project"
)

type ProjectTestSuite struct {
	suite.Suite
	projectFile *projectfile.Project
	project     *project.Project
	testdataDir string
}

func (suite *ProjectTestSuite) BeforeTest(suiteName, testName string) {
	projectfile.Reset()

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")

	suite.testdataDir = filepath.Join(root, "pkg", "project", "testdata")
	err = os.Chdir(suite.testdataDir)
	suite.Require().NoError(err, "Should change dir without issue.")
	projectFile, err := projectfile.GetSafe()
	suite.Require().NoError(err, errs.Join(err, "\n").Error())
	projectFile.Persist()
	suite.projectFile = projectFile
	suite.Require().Nil(err, "Should retrieve projectfile without issue.")
	suite.project, err = project.GetSafe()
	suite.Require().Nil(err, "Should retrieve project without issue.")
}

func (suite *ProjectTestSuite) TestGet() {
	config := project.Get()
	suite.NotNil(config, "Config should be set")
}

func (suite *ProjectTestSuite) TestGetSafe() {
	val, err := project.GetSafe()
	suite.NoError(err, "Run without failure")
	suite.NotNil(val, "Config should be set")
}

func (suite *ProjectTestSuite) TestProject() {
	suite.Equal("https://platform.activestate.com/ActiveState/project?branch=main&commitID=00010001-0001-0001-0001-000100010001", suite.project.URL(), "Values should match")
	suite.Equal("project", suite.project.Name(), "Values should match")
	suite.Equal("00010001-0001-0001-0001-000100010001", suite.project.CommitID(), "Values should match")
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

func (suite *ProjectTestSuite) TestPlatforms() {
	val := suite.project.Platforms()
	plat := val[0]
	suite.Equal(4, len(val), "Values should match")

	name := plat.Name()
	suite.Equal("fullexample", name, "Names should match")

	os, err := plat.Os()
	suite.NoError(err)
	suite.Equal("darwin", os, "OS should match")

	var version string
	suite.NoError(err)
	version, err = plat.Version()
	suite.Equal("10.0", version, "Version should match")

	var arch string
	arch, err = plat.Architecture()
	suite.NoError(err)
	suite.Equal("x386", arch, "Arch should match")

	var libc string
	libc, err = plat.Libc()
	suite.NoError(err)
	suite.Equal("gnu", libc, "Libc should match")

	var compiler string
	compiler, err = plat.Compiler()
	suite.NoError(err)
	suite.Equal("gcc", compiler, "Compiler should match")
}

func (suite *ProjectTestSuite) TestEvents() {
	events := suite.project.Events()
	suite.Equal(1, len(events), "Should match 1 out of three constrained items")

	event := events[0]
	name := event.Name()
	value, err := event.Value()
	suite.NoError(err)

	if runtime.GOOS == "linux" {
		suite.Equal("foo", name, "Names should match (Linux)")
		suite.Equal("foo Linux", value, "Value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		suite.Equal("bar", name, "Name should match (Windows)")
		suite.Equal("bar Windows", value, "Value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		suite.Equal("baz", name, "Names should match (OSX)")
		suite.Equal("baz OSX", value, "Value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestLanguages() {
	languages := suite.project.Languages()
	suite.Equal(2, len(languages), "Should match 2 out of three constrained items")

	lang := languages[0]
	name := lang.Name()
	version := lang.Version()
	id := lang.ID()
	build, err := lang.Build()
	suite.NoError(err)

	if runtime.GOOS == "linux" {
		suite.Equal("foo", name, "Names should match (Linux)")
		suite.Equal("1.1", version, "Version should match (Linux)")
		suite.Equal("foo1.1", id, "ID should match (Linux)")
		suite.Equal("--foo Linux", (*build)["override"], "Build value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		suite.Equal("bar", name, "Name should match (Windows)")
		suite.Equal("1.2", version, "Version should match (Windows)")
		suite.Equal("bar1.2", id, "ID should match (Windows)")
		suite.Equal("--bar Windows", (*build)["override"], "Build value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		suite.Equal("baz", name, "Names should match (OSX)")
		suite.Equal("1.3", version, "Version should match (OSX)")
		suite.Equal("baz1.3", id, "ID should match (OSX)")
		suite.Equal("--baz OSX", (*build)["override"], "Build value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestPackages() {
	languages := suite.project.Languages()
	var language *project.Language
	for _, l := range languages {
		if l.Name() == "packages" {
			language = l
		}
	}
	packages := language.Packages()
	suite.Equal(1, len(packages), "Should match 1 out of three constrained items")

	pkg := packages[0]
	name := pkg.Name()
	version := pkg.Version()
	build, err := pkg.Build()
	suite.NoError(err)

	if runtime.GOOS == "linux" {
		suite.Equal("foo", name, "Names should match (Linux)")
		suite.Equal("1.1", version, "Version should match (Linux)")
		suite.Equal("--foo Linux", (*build)["override"], "Build value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		suite.Equal("bar", name, "Name should match (Windows)")
		suite.Equal("1.2", version, "Version should match (Windows)")
		suite.Equal("--bar Windows", (*build)["override"], "Build value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		suite.Equal("baz", name, "Names should match (OSX)")
		suite.Equal("1.3", version, "Version should match (OSX)")
		suite.Equal("--baz OSX", (*build)["override"], "Build value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestScripts() {
	scripts := suite.project.Scripts()
	suite.Equal(1, len(scripts), "Should match 1 out of three constrained items")

	script := scripts[0]
	name := script.Name()
	value, err := script.Value()
	suite.NoError(err)
	raw := script.Raw()
	safe := script.LanguageSafe()
	standalone := script.Standalone()

	if runtime.GOOS == "linux" {
		suite.Equal("foo", name, "Names should match (Linux)")
		suite.Equal("foo Linux", value, "Value should match (Linux)")
		suite.Equal("foo $platform.name", raw, "Raw value should match (Linux)")
		suite.Equal([]language.Language{language.Sh}, safe, "Safe language should match (Linux)")
		suite.True(standalone, "Standalone value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		suite.Equal("bar", name, "Name should match (Windows)")
		suite.Equal("bar Windows", value, "Value should match (Windows)")
		suite.Equal("bar $platform.name", raw, "Raw value should match (Windows)")
		suite.Equal([]language.Language{language.Batch}, safe, "Safe language should match (Windows)")
		suite.True(standalone, "Standalone value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		suite.Equal("baz", name, "Names should match (OSX)")
		suite.Equal("baz OSX", value, "Value should match (OSX)")
		suite.Equal("baz $platform.name", raw, "Raw value should match (OSX)")
		suite.Equal([]language.Language{language.Sh}, safe, "Language should match (OSX)")
		suite.True(standalone, "Standalone value should match (OSX)")
	}
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
		suite.Equal("foo Linux", v, "Value should match (Linux)")
		suite.True(script.Standalone(), "Standalone value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		script = suite.project.ScriptByName("bar")
		suite.Require().NotNil(script)
		suite.Equal("bar", script.Name(), "Name should match (Windows)")
		v, err := script.Value()
		suite.NoError(err)
		suite.Equal("bar Windows", v, "Value should match (Windows)")
		suite.True(script.Standalone(), "Standalone value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		script = suite.project.ScriptByName("baz")
		suite.Require().NotNil(script)
		suite.Equal("baz", script.Name(), "Names should match (OSX)")
		v, err := script.Value()
		suite.NoError(err)
		suite.Equal("baz OSX", v, "Value should match (OSX)")
		suite.True(script.Standalone(), "Standalone value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestConstants() {
	constants := suite.project.Constants()

	constant := constants[0]

	name := constant.Name()
	value, err := constant.Value()
	suite.NoError(err)

	if runtime.GOOS == "linux" {
		suite.Equal("foo", name, "Names should match (Linux)")
		suite.Equal("foo Linux", value, "Value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		suite.Equal("bar", name, "Name should match (Windows)")
		suite.Equal("bar Windows", value, "Value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		suite.Equal("baz", name, "Names should match (OSX)")
		suite.Equal("baz OSX", value, "Value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestSecrets() {
	prj, err := project.GetSafe()
	suite.NoError(err, "Run without failure")
	cfg, err := config.Get()
	suite.Require().NoError(err)
	secrets := prj.Secrets(cfg)
	suite.Len(secrets, 2)

	userSecret := prj.SecretByName("secret", project.SecretScopeUser, cfg)
	suite.Require().NotNil(userSecret)
	suite.Equal("secret-user", userSecret.Description())
	suite.True(userSecret.IsUser())
	suite.False(userSecret.IsProject())
	suite.Equal("user", userSecret.Scope())

	projectSecret := prj.SecretByName("secret", project.SecretScopeProject, cfg)
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
