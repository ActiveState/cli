package runtime_test

import (
	"os"
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/sysinfo"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/stretchr/testify/require"
)

func TestDownload(t *testing.T) {
	model.OS = sysinfo.Linux
	api.UrlsByEnv["test"] = map[api.Service]string{
		api.ServicePlatform:  constants.PlatformURLDev,
		api.ServiceSecrets:   constants.SecretsURLDev,
		api.ServiceHeadChef:  constants.HeadChefURLDev,
		api.ServiceInventory: constants.InventoryURLDev,
	}
	api.DetectServiceURLs()

	download.SetMocking(false)

	fail := authentication.Get().AuthenticateWithToken(os.Getenv("API_TOKEN"))
	require.NoError(t, fail.ToError())

	pj := &projectfile.Project{Name: "ActivePython-3.5", Owner: "ActiveState"}
	r := runtime.InitRuntimeDownload(project.New(pj), "/tmp/out")
	filename, fail := r.Download()
	fmt.Sprintf("Downloaded %s", filename)
	require.NoError(t, fail.ToError())
}
