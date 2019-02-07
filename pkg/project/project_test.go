package project

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var cwd string

func setProjectDir(t *testing.T) {
	var err error
	cwd, err = environment.GetRootPath()
	assert.NoError(t, err, "Should fetch cwd")
	err = os.Chdir(filepath.Join(cwd, "pkg", "project", "testdata"))
	assert.NoError(t, err, "Should change dir without issue.")
	projectfile.Reset()
}

type ProjectTestSuite struct {
	suite.Suite

	testdataDir string

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *ProjectTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")

	suite.testdataDir = filepath.Join(root, "pkg", "project", "testdata")
	err = os.Chdir(suite.testdataDir)
	suite.Require().NoError(err, "Should change dir without issue.")

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient
	secretsClient.Persist()

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.Prefix)
}

func (suite *ProjectTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *ProjectTestSuite) prepareWorkingExpander() {
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/projects/project", 200)

	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", 200)
}

func (suite *ProjectTestSuite) TestGet() {
	config := Get()
	suite.NotNil(config, "Config should be set")
}

func (suite *ProjectTestSuite) TestGetSafe() {
	val, fail := GetSafe()
	suite.NoError(fail.ToError(), "Run without failure")
	suite.NotNil(val, "Config should be set")
}

func (suite *ProjectTestSuite) TestProject() {
	prj, fail := GetSafe()
	suite.Nil(fail, "Run without failure")
	suite.Equal("project", prj.Name(), "Values should match")
	suite.Equal("ActiveState", prj.Owner(), "Values should match")
	suite.Equal("my/name/space", prj.Namespace(), "Values should match")
	suite.Equal("something", prj.Environments(), "Values should match")
	suite.Equal("1.0", prj.Version(), "Values should match")
}

func (suite *ProjectTestSuite) TestWhenInSubDirectories() {
	err := os.Chdir(filepath.Join(suite.testdataDir, "sub1", "sub2"))
	suite.Require().NoError(err, "Should change dir without issue.")

	prj, fail := GetSafe()
	suite.Require().Nil(fail, "Run without failure")
	suite.Equal("project", prj.Name(), "Values should match")
	suite.Equal("ActiveState", prj.Owner(), "Values should match")
	suite.Equal("my/name/space", prj.Namespace(), "Values should match")
	suite.Equal("something", prj.Environments(), "Values should match")
	suite.Equal("1.0", prj.Version(), "Values should match")
}

func (suite *ProjectTestSuite) TestPlatforms() {
	prj, fail := GetSafe()
	suite.Nil(fail, "Run without failure")
	val := prj.Platforms()
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
	prj, fail := GetSafe()
	suite.NoError(fail.ToError(), "Run without failure")

	events := prj.Events()
	suite.Equal(1, len(events), "Should match 1 out of three constrained items")

	event := events[0]

	if runtime.GOOS == "linux" {
		name := event.Name()
		suite.Equal("foo", name, "Names should match (Linux)")
		value := event.Value()
		suite.Equal("foo Linux", value, "Value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := event.Name()
		suite.Equal("bar", name, "Name should match (Windows)")
		value := event.Value()
		suite.Equal("bar Windows", value, "Value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := event.Name()
		suite.Equal("baz", name, "Names should match (OSX)")
		value := event.Value()
		suite.Equal("baz OSX", value, "Value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestLanguages() {
	prj, fail := GetSafe()
	suite.Nil(fail, "Run without failure")

	languages := prj.Languages()
	suite.Equal(2, len(languages), "Should match 1 out of three constrained items")

	lang := languages[0]

	if runtime.GOOS == "linux" {
		name := lang.Name()
		suite.Equal("foo", name, "Names should match (Linux)")
		version := lang.Version()
		suite.Equal("1.1", version, "Version should match (Linux)")
		build := lang.Build()
		suite.Equal("--foo Linux", (*build)["override"], "Build value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := lang.Name()
		suite.Equal("bar", name, "Name should match (Windows)")
		version := lang.Version()
		suite.Equal("1.2", version, "Version should match (Windows)")
		build := lang.Build()
		suite.Equal("--bar Windows", (*build)["override"], "Build value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := lang.Name()
		suite.Equal("baz", name, "Names should match (OSX)")
		version := lang.Version()
		suite.Equal("1.3", version, "Version should match (OSX)")
		build := lang.Build()
		suite.Equal("--baz OSX", (*build)["override"], "Build value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestPackages() {
	prj, fail := GetSafe()
	suite.Nil(fail, "Run without failure")
	languages := prj.Languages()
	var language *Language
	for _, l := range languages {
		if l.Name() == "packages" {
			language = l
		}
	}
	packages := language.Packages()
	suite.Equal(1, len(packages), "Should match 1 out of three constrained items")

	pkg := packages[0]

	if runtime.GOOS == "linux" {
		name := pkg.Name()
		suite.Equal("foo", name, "Names should match (Linux)")
		version := pkg.Version()
		suite.Equal("1.1", version, "Version should match (Linux)")
		build := pkg.Build()
		suite.Equal("--foo Linux", (*build)["override"], "Build value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := pkg.Name()
		suite.Equal("bar", name, "Name should match (Windows)")
		version := pkg.Version()
		suite.Equal("1.2", version, "Version should match (Windows)")
		build := pkg.Build()
		suite.Equal("--bar Windows", (*build)["override"], "Build value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := pkg.Name()
		suite.Equal("baz", name, "Names should match (OSX)")
		version := pkg.Version()
		suite.Equal("1.3", version, "Version should match (OSX)")
		build := pkg.Build()
		suite.Equal("--baz OSX", (*build)["override"], "Build value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestScripts() {
	prj, fail := GetSafe()
	suite.Nil(fail, "Run without failure")
	scripts := prj.Scripts()
	suite.Equal(1, len(scripts), "Should match 1 out of three constrained items")

	script := scripts[0]

	if runtime.GOOS == "linux" {
		name := script.Name()
		suite.Equal("foo", name, "Names should match (Linux)")
		version := script.Value()
		suite.Equal("foo Linux", version, "Value should match (Linux)")
		standalone := script.Standalone()
		suite.True(standalone, "Standalone value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := script.Name()
		suite.Equal("bar", name, "Name should match (Windows)")
		version := script.Value()
		suite.Equal("bar Windows", version, "Value should match (Windows)")
		standalone := script.Standalone()
		suite.True(standalone, "Standalone value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := script.Name()
		suite.Equal("baz", name, "Names should match (OSX)")
		version := script.Value()
		suite.Equal("baz OSX", version, "Value should match (OSX)")
		standalone := script.Standalone()
		suite.True(standalone, "Standalone value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestVariables() {
	prj, fail := GetSafe()
	suite.Nil(fail, "Run without failure")
	variables := prj.Variables()

	variable := variables[0]

	if runtime.GOOS == "linux" {
		name := variable.Name()
		suite.Equal("foo", name, "Names should match (Linux)")
		value := variable.Value()
		suite.Equal("foo Linux", value, "Value should match (Linux)")
	} else if runtime.GOOS == "windows" {
		name := variable.Name()
		suite.Equal("bar", name, "Name should match (Windows)")
		value := variable.Value()
		suite.Equal("bar Windows", value, "Value should match (Windows)")
	} else if runtime.GOOS == "darwin" {
		name := variable.Name()
		suite.Equal("baz", name, "Names should match (OSX)")
		value := variable.Value()
		suite.Equal("baz OSX", value, "Value should match (OSX)")
	}
}

func (suite *ProjectTestSuite) TestSecretVariables() {
	suite.prepareWorkingExpander()

	prj, fail := GetSafe()
	suite.Nil(fail, "Run without failure")

	{
		variable := prj.VariableByName("undefined-secret")
		suite.Nil(variable.ValueOrNil(), "Should be nil as the variable has not been defined")
	}

	{
		variable := prj.VariableByName("defined-secret-org")
		suite.NotNil(variable.ValueOrNil(), "Should not be nil as the variable has been defined")
	}
}

func Test_ProjectTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectTestSuite))
}
