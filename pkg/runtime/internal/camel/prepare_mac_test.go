// +build darwin

package camel_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/implementations/camel"
)

func (suite *MetaDataTestSuite) TestMetaData_Prepare() {
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
	os.Unsetenv("PYTHONIOENCODING")
	defer func() {
		os.Setenv("PYTHONIOENCODING", originalValue)
	}()

	relBinDir := filepath.Join("Library", "Frameworks", "Python.framework", "Versions", "Current", "bin")
	relVersionedDir := filepath.Join("Library", "Frameworks", "Python.framework", "Versions", "3.7")

	// Directory that contains binary file on MacOS
	tempDir := filepath.Join(suite.dir, relBinDir)
	err := fileutils.Mkdir(tempDir)
	suite.Require().NoError(err)

	versionedDir := filepath.Join(suite.dir, relVersionedDir)
	err = fileutils.Mkdir(versionedDir)
	suite.Require().NoError(err)

	// Directory that contains site-packages on MacOS
	err = fileutils.Mkdir(suite.dir, "Library/Frameworks/Python.framework/Versions/Current/lib")
	suite.Require().NoError(err)

	pythonBinaryFilename := "python3"
	err = fileutils.Touch(filepath.Join(tempDir, pythonBinaryFilename))
	suite.Require().NoError(err)

	contents := fmt.Sprintf(template, tempDir)
	metaData, err := camel.ParseMetaData([]byte(contents))
	suite.Require().NoError(err)

	err = metaData.Prepare(suite.dir)
	suite.Require().NoError(err)
	suite.Assert().NotEmpty(metaData.Env["PYTHONIOENCODING"])

	suite.Len(metaData.TargetedRelocations, 1, "expected one targeted relocation")
	suite.Equal(camel.TargetedRelocation{
		InDir:        relBinDir,
		SearchString: "#!" + filepath.Join("/", relVersionedDir),
		Replacement:  "#!" + filepath.Join("${INSTALLDIR}", relVersionedDir),
	}, metaData.TargetedRelocations[0], suite.dir)
}
