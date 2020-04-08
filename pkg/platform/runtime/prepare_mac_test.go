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

	tempDir := suite.dir
	// Directory that contains binary file on MacOS
	tempDir = filepath.Join(suite.dir, "Library", "Frameworks", "Python.framework", "Versions", "Current", "bin")
	fail := fileutils.Mkdir(tempDir)
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
}
