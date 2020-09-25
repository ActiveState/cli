package integration

import (
	"fmt"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
)

var (
	testUser          = "test-user"
	testProject       = "test-project"
	namespace         = fmt.Sprintf("%s/%s", testUser, testProject)
	url               = fmt.Sprintf("https://%s/%s", constants.PlatformURL, namespace)
	sampleYAMLPython2 = ""
	sampleYAMLPython3 = ""
)

func init() {
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "batch"
	}
	sampleYAMLPython2 = locale.T("sample_yaml_python", map[string]interface{}{
		"Owner":    testUser,
		"Project":  testProject,
		"Shell":    shell,
		"Language": "python2",
	})
	sampleYAMLPython3 = locale.T("sample_yaml_python", map[string]interface{}{
		"Owner":    testUser,
		"Project":  testProject,
		"Shell":    shell,
		"Language": "python3",
	})
}
