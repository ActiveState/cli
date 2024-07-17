package camel

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
)

type MetaDataTestSuite struct {
	suite.Suite

	dir string
}

func (suite *MetaDataTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.dir, err = os.MkdirTemp("", "metadata-test")
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

	metaData, err := parseMetaData([]byte(contents))
	suite.Require().NoError(err)
	suite.Equal("PYTHONPATH", metaData.AffectedEnv)
	suite.Equal("/relocate", metaData.RelocationDir)
	suite.Equal("bin", metaData.BinaryLocations[0].Path)
	suite.Equal(true, metaData.BinaryLocations[0].Relative)
}

func TestMetaDataTestSuite(t *testing.T) {
	suite.Run(t, new(MetaDataTestSuite))
}
