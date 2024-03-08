package runtime

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// RefreshRuntime should be called after runtime mutations.
func RefreshRuntime(
	auth *authentication.Auth,
	out output.Outputer,
	an analytics.Dispatcher,
	proj *project.Project,
	commitID strfmt.UUID,
	changed bool,
	trigger target.Trigger,
	svcm *model.SvcModel,
	cfg Configurable,
) (rerr error) {
	target := target.NewProjectTarget(proj, &commitID, trigger)
	rt, err := runtime.New(target, an, svcm, auth, cfg, out)
	if err != nil {
		return locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	if !changed && !rt.NeedsUpdate() {
		out.Notice(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return nil
	}

	if rt.NeedsUpdate() {
		if !rt.HasCache() {
			out.Notice(output.Title(locale.T("install_runtime")))
			out.Notice(locale.T("install_runtime_info"))
		} else {
			out.Notice(output.Title(locale.T("update_runtime")))
			out.Notice(locale.T("update_runtime_info"))
		}
	}

	return RefreshRuntimeByReference(rt, auth, out, proj, cfg)
}

// RefreshRuntimeByReference will update the given runtime if necessary. Unlike RefreshRuntime this won't print any UI
// except for the progress of sourcing the runtime.
func RefreshRuntimeByReference(
	rt *runtime.Runtime,
	auth *authentication.Auth,
	out output.Outputer,
	proj *project.Project,
	cfg Configurable,
) (rerr error) {
	if cfg.GetBool(constants.OptinBuildscriptsConfig) {
		_, err := buildscript.Sync(proj, ptr.To(rt.Target().CommitUUID()), out, auth)
		if err != nil {
			return locale.WrapError(err, "err_update_build_script")
		}
	}

	if rt.NeedsUpdate() {
		pg := runbits.NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := rt.SolveAndUpdate(pg)
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return nil
}

// UpdateByReference will update the given runtime if necessary. This is functionally the same as RefreshRuntimeByReference
// except that it does not do its own solve.
func UpdateByReference(
	rt *runtime.Runtime,
	buildResult *model.BuildResult,
	auth *authentication.Auth,
	out output.Outputer,
	proj *project.Project,
	cfg Configurable,
) (rerr error) {
	if cfg.GetBool(constants.OptinBuildscriptsConfig) {
		_, err := buildscript.Sync(proj, ptr.To(rt.Target().CommitUUID()), out, auth)
		if err != nil {
			return locale.WrapError(err, "err_update_build_script")
		}
	}

	if rt.NeedsUpdate() {
		pg := runbits.NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := rt.Setup(pg).Update(buildResult)
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return nil
}
