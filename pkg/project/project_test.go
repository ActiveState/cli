package project_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/stretchr/testify/suite"
)

type ProjectTestSuite struct {
	suite.Suite
	projectFile *projectfile.Project
	project     *project.Project
	testdataDir string
}

func (suite *ProjectTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	projectfile.Reset()

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")

	suite.testdataDir = filepath.Join(root, "pkg", "project", "testdata")
	err = os.Chdir(suite.testdataDir)
	suite.Require().NoError(err, "Should change dir without issue.")
	projectFile, fail := projectfile.GetSafe()
	projectFile.Persist()
	suite.projectFile = projectFile
	suite.Require().Nil(fail, "Should retrieve projectfile without issue.")
	suite.project, fail = project.GetSafe()
	suite.Require().Nil(fail, "Should retrieve project without issue.")
}

func (suite *ProjectTestSuite) TestGet() {
	config := project.Get()
	suite.NotNil(config, "Config should be set")
}

func (suite *ProjectTestSuite) TestGetSafe() {
	val, fail := project.GetSafe()
	suite.NoError(fail.ToError(), "Run without failure")
	suite.NotNil(val, "Config should be set")
}

func (suite *ProjectTestSuite) TestProject() {
	suite.Equal("https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001", suite.project.URL(), "Values should match")
	suite.Equal("project", suite.project.Name(), "Values should match")
	suite.Equal("00010001-0001-0001-0001-000100010001", suite.project.CommitID(), "Values should match")
	suite.Equal("ActiveState", suite.project.Owner(), "Values should match")
	suite.Equal("ActiveState/project", suite.project.Namespace(), "Values should match")
	suite.Equal("my/name/space", suite.project.Source().Namespace, "Values should match")
	suite.Equal("something", suite.project.Environments(), "Values should match")
	suite.Equal("1.0", suite.project.Version(), "Values should match")
}

func (suite *ProjectTestSuite) TestWhenInSubDirectories() {
	err := os.Chdir(filepath.Join(suite.testdataDir, "sub1", "sub2"))
	suite.Require().NoError(err, "Should change dir without issue.")

	suite.Equal("project", suite.project.Name(), "Values should match")
	suite.Equal("ActiveState", suite.project.Owner(), "Values should match")
	suite.Equal("ActiveState/project", suite.project.Namespace(), "Values should match")
	suite.Equal("my/name/space", suite.project.Source().Namespace, "Values should match")
	suite.Equal("something", suite.project.Environments(), "Values should match")
	suite.Equal("1.0", suite.project.Version(), "Values should match")
}

func (suite *ProjectTestSuite) TestPlatforms() {
	val := suite.project.Platforms()
	plat := val[0]
	suite.Equal(4, len(val), "Values should match")

	name := plat.Name()
	suite.Equal("fullexample", name, "Names should match")

	os := plat.Os()
	suite.Equal("darwin", os, "OS should match")

	var version string
	version = plat.Version()
	suite.Equal("10.0", version, "Version should match")

	var arch string
	arch = plat.Architecture()
	suite.Equal("x386", arch, "Arch should match")

	var libc string
	libc = plat.Libc()
	suite.Equal("gnu", libc, "Libc should match")

	var compiler string
	compiler = plat.Compiler()
	suite.Equal("gcc", compiler, "Compiler should match")
}

func (suite *ProjectTestSuite) TestEvents() {
	events := suite.project.Events()
	suite.Equal(1, len(events), "Should match 1 out of three constrained items")

	event := events[0]
	name := event.Name()
	value := event.Value()

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
	build := lang.Build()

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
	build := pkg.Build()

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
	value := script.Value()
	raw := script.Raw()
	safe := script.LanguageSafe()
	standalone := script.Standalone()

	if runtime.GOOS == "linux" {
		suite.Equal("foo", name, "Names should match (Linux)")
		suite.Equal("foo Linux", value, "Value should match (Linux)")
		suite.Equal("foo $platform.name", raw, "Raw value should match (Linux)")
		suite.Equal(language.Sh, safe, "Safe language should match (Linux)")
		suite.True(standalone, "Standalone value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		suite.Equal("bar", name, "Name should match (Windows)")
		suite.Equal("bar Windows", value, "Value should match (Windows)")
		suite.Equal("bar $platform.name", raw, "Raw value should match (Windows)")
		suite.Equal(language.Batch, safe, "Safe language should match (Windows)")
		suite.True(standalone, "Standalone value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		suite.Equal("baz", name, "Names should match (OSX)")
		suite.Equal("baz OSX", value, "Value should match (OSX)")
		suite.Equal("baz $platform.name", raw, "Raw value should match (OSX)")
		suite.Equal(language.Sh, safe, "Language should match (OSX)")
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
		suite.Equal("foo Linux", script.Value(), "Value should match (Linux)")
		suite.True(script.Standalone(), "Standalone value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		script = suite.project.ScriptByName("bar")
		suite.Require().NotNil(script)
		suite.Equal("bar", script.Name(), "Name should match (Windows)")
		suite.Equal("bar Windows", script.Value(), "Value should match (Windows)")
		suite.True(script.Standalone(), "Standalone value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		script = suite.project.ScriptByName("baz")
		suite.Require().NotNil(script)
		suite.Equal("baz", script.Name(), "Names should match (OSX)")
		suite.Equal("baz OSX", script.Value(), "Value should match (OSX)")
		suite.True(script.Standalone(), "Standalone value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestConstants() {
	constants := suite.project.Constants()

	constant := constants[0]

	name := constant.Name()
	value := constant.Value()

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
	prj, fail := project.GetSafe()
	suite.NoError(fail.ToError(), "Run without failure")
	secrets := prj.Secrets()
	suite.Len(secrets, 2)

	userSecret := prj.SecretByName("secret", project.SecretScopeUser)
	suite.Require().NotNil(userSecret)
	suite.Equal("secret-user", userSecret.Description())
	suite.True(userSecret.IsUser())
	suite.False(userSecret.IsProject())
	suite.Equal("user", userSecret.Scope())

	projectSecret := prj.SecretByName("secret", project.SecretScopeProject)
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
