package runbits

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// RefreshRuntime should be called after runtime mutations. A nil arg for "req"
// means that the message handler will not print output for "a single
// requirement". For example, if multiple requirements are affected, nil is the
// appropriate value.
func RefreshRuntime(out output.Outputer, req *RequestedRequirement, proj *project.Project, cachePath string, commitID strfmt.UUID, changed bool) error {
	rtMessages := NewRuntimeMessageHandler(out)
	if req != nil {
		rtMessages.SetRequirement(req)
	}
	rt, err := runtime.NewRuntime(proj.Source().Path(), cachePath, commitID, proj.Owner(), proj.Name(), rtMessages)
	if err != nil {
		return locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	if !changed && rt.IsCachedRuntime() {
		out.Print(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return nil
	}

	if !rt.IsCachedRuntime() {
		out.Notice(output.Heading(locale.Tl("update_runtime", "Updating Runtime")))
		out.Notice(locale.Tl("update_runtime_info", "Changes to your runtime may require some dependencies to be rebuilt."))
		_, _, err := runtime.NewInstaller(rt).Install()
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return nil
}
