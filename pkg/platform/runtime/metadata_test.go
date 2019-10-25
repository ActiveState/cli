package runtime_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/fileutils"
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

func TestHasBinaryFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	pythonBinaryFilename := "python3"
	binaryFile, fail := fileutils.Touch(filepath.Join(tempDir, pythonBinaryFilename))
	fmt.Println("binary file: ", binaryFile.Name())
	require.NoError(t, fail.ToError())

	pythonBinary := runtime.MetaDataBinary{
		Path:     tempDir,
		Relative: false,
	}

	meta := &runtime.MetaData{
		BinaryLocations: []runtime.MetaDataBinary{pythonBinary},
	}
	require.True(t, meta.HasBinaryFile(pythonBinaryFilename))
}
