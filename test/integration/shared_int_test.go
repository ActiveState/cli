package integration

import (
	"fmt"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
)

var (
	testUser    = "test-user"
	testProject = "test-project"
	namespace   = fmt.Sprintf("%s/%s", testUser, testProject)
	url         = fmt.Sprintf("https://%s/%s", constants.PlatformURL, namespace)
	sampleYAML  = ""
)

func init() {
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "batch"
	}
	sampleYAML = locale.T("sample_yaml", map[string]interface{}{
		"Owner":   testUser,
		"Project": testProject,
		"Shell":   shell,
	})
}
