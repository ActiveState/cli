package runbits

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// RefreshRuntime should be called after runtime mutations.
func RefreshRuntime(auth *authentication.Auth, out output.Outputer, an analytics.Dispatcher, proj *project.Project, commitID strfmt.UUID, changed bool, trigger target.Trigger, svcm *model.SvcModel) (rerr error) {
	err := buildscript.Sync(proj, &commitID, out, auth)
	if err != nil {
		return locale.WrapError(err, "err_update_build_script")
	}
	target := target.NewProjectTarget(proj, nil, trigger) // buildscript.Sync updates project's commit ID until DX-1852
	isCached := true
	rt, err := runtime.New(target, an, svcm)
	if err != nil {
		if runtime.IsNeedsUpdateError(err) {
			isCached = false
		} else {
			return locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
		}
	}

	if !changed && isCached {
		out.Notice(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return nil
	}

	if !isCached {
		if !rt.HasCache() {
			out.Notice(output.Title(locale.Tl("install_runtime", "Installing Runtime")))
			out.Notice(locale.Tl("install_runtime_info", "Installing your runtime and dependencies."))
		} else {
			out.Notice(output.Title(locale.Tl("update_runtime", "Updating Runtime")))
			out.Notice(locale.Tl("update_runtime_info", "Changes to your runtime may require some dependencies to be rebuilt.\n"))
		}
		pg := NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := rt.Update(auth, pg)
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return nil
}
