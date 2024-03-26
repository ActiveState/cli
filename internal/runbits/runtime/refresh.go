package runtime

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
	"github.com/imacks/bitflags-go"
)

type Opts int

const (
	OptNone Opts = 1 << iota
	OptMinimalUI
	OptOrderChanged
)

// SolveAndUpdate should be called after runtime mutations.
func SolveAndUpdate(
	auth *authentication.Auth,
	out output.Outputer,
	an analytics.Dispatcher,
	proj *project.Project,
	customCommitID *strfmt.UUID,
	trigger target.Trigger,
	svcm *model.SvcModel,
	cfg Configurable,
	opts Opts,
) (_ *runtime.Runtime, rerr error) {
	defer rationalizeError(auth, proj, &rerr)

	if proj == nil {
		return nil, rationalize.ErrNoProject
	}

	if proj.IsHeadless() {
		return nil, rationalize.ErrHeadless
	}

	target := target.NewProjectTarget(proj, customCommitID, trigger)
	rt, err := runtime.New(target, an, svcm, auth, cfg, out)
	if err != nil {
		return nil, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	if !bitflags.Has(opts, OptOrderChanged) && !bitflags.Has(opts, OptMinimalUI) && !rt.NeedsUpdate() {
		out.Notice(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return rt, nil
	}

	if rt.NeedsUpdate() && !bitflags.Has(opts, OptMinimalUI) {
		if !rt.HasCache() {
			out.Notice(output.Title(locale.T("install_runtime")))
			out.Notice(locale.T("install_runtime_info"))
		} else {
			out.Notice(output.Title(locale.T("update_runtime")))
			out.Notice(locale.T("update_runtime_info"))
		}
	}

	if rt.NeedsUpdate() {
		pg := NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := rt.SolveAndUpdate(pg)
		if err != nil {
			return nil, locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return rt, nil
}

// UpdateByReference will update the given runtime if necessary. This is functionally the same as SolveAndUpdateByReference
// except that it does not do its own solve.
func UpdateByReference(
	rt *runtime.Runtime,
	buildResult *model.BuildResult,
	auth *authentication.Auth,
	proj *project.Project,
	out output.Outputer,
) (rerr error) {
	defer rationalizeError(auth, proj, &rerr)

	if rt.NeedsUpdate() {
		pg := NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := rt.Setup(pg).Update(buildResult)
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return nil
}
