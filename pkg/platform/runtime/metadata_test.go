package runtime_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type MetaDataTestSuite struct {
	suite.Suite

	dir string
}

func (suite *MetaDataTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.dir, err = ioutil.TempDir("", "metadata-test")
	suite.Require().NoError(err)
}

func (suite *MetaDataTestSuite) AfterTest(suiteName, testName string) {
	err := os.RemoveAll(suite.dir)
	suite.Require().NoError(err)
}

func (suite *MetaDataTestSuite) TestMetaData() {
	contents := `{
		"affected_env": "PYTHONPATH",
		"binaries_in": [
			{
				"path": "bin",
				"relative": 1
			}
		],
		"relocation_dir": "/relocate"
	}`

	metaData, fail := runtime.ParseMetaData([]byte(contents))
	suite.Require().NoError(fail.ToError())
	suite.Equal("PYTHONPATH", metaData.AffectedEnv)
	suite.Equal("/relocate", metaData.RelocationDir)
	suite.Equal("bin", metaData.BinaryLocations[0].Path)
	suite.Equal(true, metaData.BinaryLocations[0].Relative)
}

func (suite *MetaDataTestSuite) TestMetaData_MakeBackwardsCompatible() {
	template := `{
		"affected_env": "PYTHONPATH",
		"binaries_in": [
			{
				"path": "%s",
				"relative": 1
			}
		],
		"relocation_dir": "/relocate"
	}`

	originalValue := os.Getenv("PYTHONIOENCODING")
	defer func() {
		os.Setenv("PYTHONIOENCODING", originalValue)
	}()

	tempDir := suite.dir
	pythonBinaryFilename := "python3"
	if rt.GOOS == "windows" {
		pythonBinaryFilename = pythonBinaryFilename + ".exe"
		tempDir = strings.ReplaceAll(tempDir, "\\", "\\\\")
	}
	tempBinary, fail := fileutils.Touch(filepath.Join(suite.dir, pythonBinaryFilename))
	suite.Require().NoError(fail.ToError())
	defer tempBinary.Close()

	if rt.GOOS == "darwin" {
		fail := fileutils.Mkdir("Library/Frameworks/Python.framework/Versions/Current/lib")
		suite.Require().NoError(fail.ToError())
	}

	contents := fmt.Sprintf(template, tempDir)
	metaData, fail := runtime.ParseMetaData([]byte(contents))
	suite.Require().NoError(fail.ToError())

	fail = metaData.MakeBackwardsCompatible()
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(metaData.Env["PYTHONIOENCODING"])
}

func TestMetaDataTestSuite(t *testing.T) {
	suite.Run(t, new(MetaDataTestSuite))
}
