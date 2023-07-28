package runtime

import (
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits"
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
	auth *authentication.Auth) (_ *rt.Runtime, rerr error) {

	rti, err := rt.New(target.NewProjectTarget(proj, nil, trigger), an, svcModel, auth)
	switch {
	case err == nil:
		break

	case rt.IsNeedsUpdateError(err):
		pg := runbits.NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		if err = rti.Update(pg); err != nil {
			if errs.Matches(err, &model.ErrOrderAuth{}) {
				return nil, errs.AddTips(
					locale.WrapInputError(err, "err_update_auth", "Could not update runtime"),
					locale.T("tip_private_project_auth"))
			}
			if errs.Matches(err, &model.ErrNoMatchingPlatform{}) {
				branches, err := model.BranchNamesForProjectFiltered(proj.Owner(), proj.Name(), proj.BranchName())
				if err == nil && len(branches) > 1 {
					return nil, locale.NewInputError("err_alternate_branches", "", proj.BranchName(), strings.Join(branches, "\n - "))
				}
			}
			if !auth.Authenticated() {
				return nil, errs.AddTips(
					locale.WrapError(err, "err_new_runtime_auth", "Could not update runtime installation."),
					locale.T("tip_private_project_auth"))
			}
			return nil, locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")
		}

	case rt.IsNeedsCommitError(err):
		out.Notice(locale.T("notice_commit_build_script"))

	default:
		return nil, locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
	}

	return rti, nil
}
