package integration

import (
	"fmt"

	"github.com/ActiveState/cli/internal/constants"
)

var (
	testUser    = "test-user"
	testProject = "test-project"
	namespace   = fmt.Sprintf("%s/%s", testUser, testProject)
	url         = fmt.Sprintf("https://%s/%s", constants.PlatformURL, namespace)
)
