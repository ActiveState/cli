package runtime

import (
	"strings"

	"github.com/ActiveState/cli/internal-as/analytics"
	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/runbits"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	rt "github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

// NewFromProject is a helper function that creates a new runtime or updates an existing one for
// the given project.
func NewFromProject(
	proj *project.Project,
	trigger target.Trigger,
	an analytics.Dispatcher,
	svcModel *model.SvcModel,
	out output.Outputer,
	auth *authentication.Auth) (*rt.Runtime, error) {
	projectTarget := target.NewProjectTarget(proj, storage.CachePath(), nil, trigger)
	rti, err := rt.New(projectTarget, an, svcModel)
	if err != nil {
		if !rt.IsNeedsUpdateError(err) {
			return nil, locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
		}

		eh, err := runbits.ActivateRuntimeEventHandler(out)
		if err != nil {
			return nil, locale.WrapError(err, "err_initialize_runtime_event_handler")
		}

		if err = rti.Update(auth, eh); err != nil {
			if errs.Matches(err, &model.ErrOrderAuth{}) {
				return nil, locale.WrapInputError(err, "err_update_auth", "Could not update runtime, if this is a private project you may need to authenticate with `[ACTIONABLE]state auth[/RESET]`")
			}
			if errs.Matches(err, &model.ErrNoMatchingPlatform{}) {
				branches, err := model.BranchNamesForProjectFiltered(proj.Owner(), proj.Name(), proj.BranchName())
				if err == nil && len(branches) > 1 {
					return nil, locale.NewInputError("err_alternate_branches", "", proj.BranchName(), strings.Join(branches, "\n - "))
				}
			}
			if !auth.Authenticated() {
				return nil, locale.WrapError(err, "err_new_runtime_auth", "Could not update runtime installation. If this is a private project ensure that you are authenticated.")
			}
			return nil, locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")
		}
	}
	return rti, nil
}
