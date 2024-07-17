//go:build !darwin
// +build !darwin

package camel

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
)

func (suite *MetaDataTestSuite) TestMetaData_Prepare() {
	template := `{
		"affected_env": "PYTHONPATH",
		"binaries_in": [
			{
				"path": "%s",
				"relative": 0
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
	err := fileutils.Touch(filepath.Join(suite.dir, pythonBinaryFilename))
	suite.Require().NoError(err)

	contents := fmt.Sprintf(template, tempDir)
	metaData, err := parseMetaData([]byte(contents))
	suite.Require().NoError(err)

	err = metaData.Prepare(suite.dir)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(metaData.Env["PYTHONIOENCODING"])
}
