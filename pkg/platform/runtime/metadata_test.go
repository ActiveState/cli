package runtime_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type MetaDataTestSuite struct {
	suite.Suite
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

func TestMetaDataTestSuite(t *testing.T) {
	suite.Run(t, new(MetaDataTestSuite))
}
