package runtime_test

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/stretchr/testify/require"
)

func TestDownload(t *testing.T) {
	fail := authentication.Get().AuthenticateWithToken(os.Getenv("API_TOKEN"))
	require.NoError(t, fail.ToError())

	r := runtime.InitRuntimeDownload(project.Get())
	fail = r.Download()
	require.NoError(t, fail.ToError())
}
