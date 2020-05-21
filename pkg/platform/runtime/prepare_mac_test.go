// +build darwin

package runtime_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
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
	fail := fileutils.Mkdir(tempDir)
	suite.Require().NoError(fail.ToError())

	versionedDir := filepath.Join(suite.dir, relVersionedDir)
	fail = fileutils.Mkdir(versionedDir)
	suite.Require().NoError(fail.ToError())

	// Directory that contains site-packages on MacOS
	fail = fileutils.Mkdir(suite.dir, "Library/Frameworks/Python.framework/Versions/Current/lib")
	suite.Require().NoError(fail.ToError())

	pythonBinaryFilename := "python3"
	fail = fileutils.Touch(filepath.Join(tempDir, pythonBinaryFilename))
	suite.Require().NoError(fail.ToError())

	contents := fmt.Sprintf(template, tempDir)
	metaData, fail := runtime.ParseMetaData([]byte(contents))
	metaData.Path = suite.dir
	suite.Require().NoError(fail.ToError())

	fail = metaData.Prepare()
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(metaData.Env["PYTHONIOENCODING"])

	suite.Len(metaData.TargetedRelocations, 1, "expected one targeted relocation")
	suite.Equal(runtime.TargetedRelocation{
		InDir:        tempDir,
		SearchString: "#!" + filepath.Join("/", relVersionedDir),
		Replacement:  "#!" + versionedDir,
	}, metaData.TargetedRelocations[0], suite.dir)
}
