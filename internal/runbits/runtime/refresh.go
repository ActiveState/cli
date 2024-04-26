package runtime

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

func init() {
	configMediator.RegisterOption(constants.AsyncRuntimeConfig, configMediator.Bool, false)
}

type Opts int

const (
	OptNone         Opts = 1 << iota
	OptMinimalUI         // Only print progress output, don't decorate the UI in any other way
	OptOrderChanged      // Indicate that the order has changed, and the runtime should be refreshed regardless of internal dirty checking mechanics
)

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}

// UpdateByReference will update the given runtime if necessary. This is functionally the same as SolveAndUpdateByReference
// except that it does not do its own solve.
func UpdateByReference(
	rt *runtime.Runtime,
	buildResult *model.BuildResult,
	commit *bpModel.Commit,
	auth *authentication.Auth,
	proj *project.Project,
	out output.Outputer,
) (rerr error) {
	defer rationalizeError(auth, proj, &rerr)

	if rt.NeedsUpdate() {
		pg := NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := rt.Setup(pg).Update(buildResult, commit)
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return nil
}
