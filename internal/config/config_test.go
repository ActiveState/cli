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

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
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
	dir, err := ioutil.TempDir("", "cli-config-test")
	suite.Require().NoError(err)

	viper.Reset()
	suite.config = config.New(dir)
	suite.config.Exit = exiter.Exit
}

func (suite *ConfigTestSuite) AfterTest(suiteName, testName string) {
}

func (suite *ConfigTestSuite) TestConfig() {
	suite.NotEmpty(config.ConfigPath())
	suite.NotEmpty(config.CachePath())
}

func (suite *ConfigTestSuite) TestIncludesBranch() {
	cfg := config.New("")
	suite.Contains(cfg.ConfigPath(), filepath.Clean(constants.BranchName))
}

func (suite *ConfigTestSuite) TestFilesExist() {
	suite.FileExists(filepath.Join(suite.config.ConfigPath(), suite.config.Filename()))
	suite.DirExists(filepath.Join(suite.config.CachePath()))
}

func (suite *ConfigTestSuite) TestCorruption() {
	path := filepath.Join(suite.config.ConfigPath(), suite.config.Filename())
	fail := fileutils.WriteFile(path, []byte("&"))
	suite.Require().NoError(fail.ToError())

	exiter := exiter.New()
	suite.config.Exit = exiter.Exit
	viper.Reset()

	exitCode := exiter.WaitForExit(func() {
		suite.config.ReadInConfig()
	})

	suite.Equal(1, exitCode, "Config should fail to parse")
}

// testNoHomeRunner will run the TestNoHome test in its own process, this is because the configdir package we use
// interprets the HOME env var at init time, so we cannot spoof it any other way besides when running the got test command
// and we don't want tests that require special knowledge of how to invoke them
func (suite *ConfigTestSuite) testNoHomeRunner() {
	pkgPath := reflect.TypeOf(*suite.config).PkgPath()
	args := []string{"test", pkgPath, "-run", "TestConfigTestSuite", "-testify.m", "TestNoHome"}
	fmt.Printf("Executing: go %s", strings.Join(args, " "))

	goCache := os.Getenv("GOCACHE")
	if goCache == "" {
		var err error
		goCache, err = ioutil.TempDir("", "go-cache")
		suite.Require().NoError(err)
	}

	runCmd := exec.Command("go", args...)
	runCmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
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

	err := runCmd.Run()
	suite.Require().NoError(err, "Should run without error, but returned: \n### START ###\n %s\n### END ###", out.String())
}

func (suite *ConfigTestSuite) TestNoHome() {
	if os.Getenv("TESTNOHOME") == "" {
		// configfile reads our home dir at init, so we need to get creative
		suite.testNoHomeRunner()
		return
	}

	viper.Reset()
	suite.config = config.New("")

	suite.Contains(suite.config.ConfigPath(), os.TempDir())

	suite.FileExists(filepath.Join(suite.config.ConfigPath(), suite.config.Filename()))
	suite.DirExists(filepath.Join(suite.config.CachePath()))
}

func (suite *ConfigTestSuite) TestSave() {
	path := filepath.Join(suite.config.ConfigPath(), suite.config.Filename())

	viper.Set("Foo", "bar")
	config.Save()

	dat, err := ioutil.ReadFile(path)
	suite.Require().NoError(err)

	suite.Contains(string(dat), "foo: bar", "Config should contain our newly added field")
}

func (suite *ConfigTestSuite) TestSaveMerge() {
	path := filepath.Join(suite.config.ConfigPath(), suite.config.Filename())

	fail := fileutils.WriteFile(path, []byte("ishould: exist"))
	suite.Require().NoError(fail.ToError())

	viper.Set("Foo", "bar")
	config.Save()

	dat, err := ioutil.ReadFile(path)
	suite.Require().NoError(err)

	suite.Contains(string(dat), "foo: bar", "Config should contain our newly added field")
	suite.Contains(string(dat), "ishould: exist", "Config should contain the pre-existing field")
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
