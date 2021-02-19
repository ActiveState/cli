package config_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
)

type InstanceMock struct {
	config.Instance
}

func (i *InstanceMock) Name() string {
	return "cli-"
}

type ConfigTestSuite struct {
	suite.Suite
	config *config.Instance
}

func (suite *ConfigTestSuite) SetupTest() {
}

func (suite *ConfigTestSuite) BeforeTest(suiteName, testName string) {

	var err error
	suite.config, err = config.New()
	suite.Require().NoError(err)
}

func (suite *ConfigTestSuite) AfterTest(suiteName, testName string) {
}

func (suite *ConfigTestSuite) TestConfig() {
	suite.NotEmpty(suite.config.ConfigPath())
	suite.NotEmpty(suite.config.CachePath())
}

func (suite *ConfigTestSuite) TestIncludesBranch() {
	cfg, err := config.NewWithDir("")
	suite.Require().NoError(err)
	suite.Contains(cfg.ConfigPath(), filepath.Clean(constants.BranchName))
}

func (suite *ConfigTestSuite) TestFilesExist() {
	suite.FileExists(filepath.Join(suite.config.ConfigPath(), suite.config.Filename()))
	suite.DirExists(filepath.Join(suite.config.CachePath()))
}

func (suite *ConfigTestSuite) TestCorruption() {
	path := filepath.Join(suite.config.ConfigPath(), suite.config.Filename())
	err := fileutils.WriteFile(path, []byte("&"))
	suite.Require().NoError(err)

	err = suite.config.ReadInConfig()
	suite.Require().Error(err)
}

// testNoHomeRunner will run the TestNoHome test in its own process, this is because the configdir package we use
// interprets the HOME env var at init time, so we cannot spoof it any other way besides when running the go test command
// and we don't want tests that require special knowledge of how to invoke them
func (suite *ConfigTestSuite) testNoHomeRunner() {
	pkgPath := reflect.TypeOf(*suite.config).PkgPath()
	args := []string{"test", pkgPath, "-run", "TestConfigTestSuite", "-testify.m", "TestNoHome"}
	fmt.Printf("Executing: go %s", strings.Join(args, " "))

	var err error
	goCache, err := ioutil.TempDir("", "go-cache")
	suite.Require().NoError(err)

	goPath := filepath.Join(os.Getenv("GOROOT"), "GOHOME")
	if os.Getenv("GOPATH") != "" {
		goPath = os.Getenv("GOPATH")
	}

	runCmd := exec.Command("go", args...)
	runCmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"GOROOT=" + os.Getenv("GOROOT"),
		"GOENV=" + os.Getenv("GOENV"),
		"GOPATH=" + goPath,
		"USERPROFILE=" + os.Getenv("USERPROFILE"), // Permission error trying to use C:\Windows, ref: https://golang.org/pkg/os/#TempDir
		"APPDATA=" + os.Getenv("APPDATA"),
		"SystemRoot=" + os.Getenv("SystemRoot"), // Ref: https://bugs.python.org/msg248951
		"GOFLAGS=" + os.Getenv("GOFLAGS"),
		"GOCACHE=" + goCache,
		"TESTNOHOME=TRUE",
	}

	var out bytes.Buffer
	runCmd.Stdout = &out
	runCmd.Stderr = &out

	err = runCmd.Run()
	suite.Require().NoError(err, "Should run without error, but returned: \n### START ###\n %s\n### END ###", out.String())
}

func (suite *ConfigTestSuite) TestNoHome() {
	if os.Getenv("TESTNOHOME") == "" {
		// configfile reads our home dir at init, so we need to get creative
		suite.testNoHomeRunner()
		return
	}

	var err error
	suite.config, err = config.New()
	suite.Require().NoError(err)

	suite.Contains(suite.config.ConfigPath(), os.TempDir())

	suite.FileExists(filepath.Join(suite.config.ConfigPath(), suite.config.Filename()))
	suite.DirExists(filepath.Join(suite.config.CachePath()))
}

func (suite *ConfigTestSuite) TestSave() {
	path := filepath.Join(suite.config.ConfigPath(), suite.config.Filename())

	suite.config.Set("Foo", "bar")

	dat, err := ioutil.ReadFile(path)
	suite.Require().NoError(err)

	suite.Contains(string(dat), "foo: bar", "Config should contain our newly added field")
}

func (suite *ConfigTestSuite) TestSaveMerge() {
	path := filepath.Join(suite.config.ConfigPath(), suite.config.Filename())

	err := fileutils.WriteFile(path, []byte("ishould: exist"))
	suite.Require().NoError(err)

	suite.config.Set("Foo", "bar")

	dat, err := ioutil.ReadFile(path)
	suite.Require().NoError(err)

	suite.Contains(string(dat), "foo: bar", "Config should contain our newly added field")
	suite.Contains(string(dat), "ishould: exist", "Config should contain the pre-existing field")
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
