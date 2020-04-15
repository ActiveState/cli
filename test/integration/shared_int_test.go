package integration

import (
	"fmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
)

var (
	testUser    = "test-user"
	testProject = "test-project"
	namespace   = fmt.Sprintf("%s/%s", testUser, testProject)
	url         = fmt.Sprintf("https://%s/%s", constants.PlatformURL, namespace)
	sampleYAML  = locale.T("sample_yaml", map[string]interface{}{
		"Owner":   testUser,
		"Project": testProject,
	})
)
