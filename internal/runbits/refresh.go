package runbits

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// RefreshRuntime should be called after runtime mutations.
func RefreshRuntime(auth *authentication.Auth, out output.Outputer, an analytics.Dispatcher, proj *project.Project, cachePath string, commitID strfmt.UUID, changed bool, trigger target.Trigger) error {
	rtMessages, err := DefaultRuntimeEventHandler(out)
	if err != nil {
		return locale.WrapError(err, "err_initialize_runtime_event_handler")
	}
	target := target.NewProjectTarget(proj, cachePath, &commitID, trigger)
	isCached := true
	rt, err := runtime.New(target, an)
	if err != nil {
		if runtime.IsNeedsUpdateError(err) {
			isCached = false
		} else {
			return locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
		}
	}

	if !changed && isCached {
		out.Print(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return nil
	}

	if !isCached {
		if !fileutils.DirExists(target.Dir()) {
			out.Notice(output.Heading(locale.Tl("install_runtime", "Installing Runtime")))
			out.Notice(locale.Tl("install_runtime_info", "Installing your runtime and dependencies."))
		} else {
			out.Notice(output.Heading(locale.Tl("update_runtime", "Updating Runtime")))
			out.Notice(locale.Tl("update_runtime_info", "Changes to your runtime may require some dependencies to be rebuilt."))
		}
		err := rt.Update(auth, rtMessages)
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return nil
}
