// +build !darwin

package runtime_test

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

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
	pythonBinaryFilename := "python3"
	if rt.GOOS == "windows" {
		pythonBinaryFilename = pythonBinaryFilename + ".exe"
		tempDir = strings.ReplaceAll(tempDir, "\\", "\\\\")
	}
	fail := fileutils.Touch(filepath.Join(suite.dir, pythonBinaryFilename))
	suite.Require().NoError(fail)

	contents := fmt.Sprintf(template, tempDir)
	metaData, fail := runtime.ParseMetaData([]byte(contents))
	suite.Require().NoError(fail)

	fail = metaData.Prepare()
	suite.Require().NoError(fail)
	suite.Require().NotEmpty(metaData.Env["PYTHONIOENCODING"])
}
