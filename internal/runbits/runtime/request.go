package runtime

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
	"github.com/imacks/bitflags-go"
)

// TODO: An alternate approach is to create a new RuntimeDispatcher struct that
// can be used to dispatch the runtime as well as request information about the
// runtime and whether it was updated or not.
// Taking this further it could return a wrapper around the runtime object
// that can provide information about the runtime and whether it was updated or not.

// Another idea is for this wrapper to be a "RuntimeDispather" that all of the runners
// use to get their runtime. This can centralize checks like the async_runtime check
// and the NeedsUpdate check.

type Request struct {
	Auth           *authentication.Auth
	Out            output.Outputer
	An             analytics.Dispatcher
	Proj           *project.Project
	CustomCommitID *strfmt.UUID
	Trigger        target.Trigger
	SvcModel       *model.SvcModel
	Cfg            Configurable
	Opts           Opts
	asyncRuntime   bool
}

func NewRequest(auth *authentication.Auth,
	an analytics.Dispatcher,
	proj *project.Project,
	customCommitID *strfmt.UUID,
	trigger target.Trigger,
	svcm *model.SvcModel,
	cfg Configurable,
	opts Opts,
) (_ *Request, rerr error) {
	defer rationalizeError(auth, proj, &rerr)

	if proj == nil {
		return nil, rationalize.ErrNoProject
	}

	if proj.IsHeadless() {
		return nil, rationalize.ErrHeadless
	}

	return &Request{
		Auth:           auth,
		An:             an,
		Proj:           proj,
		CustomCommitID: customCommitID,
		Trigger:        trigger,
		SvcModel:       svcm,
		Cfg:            cfg,
		Opts:           opts,
		asyncRuntime:   cfg.GetBool(constants.AsyncRuntimeConfig),
	}, nil
}

func (r *Request) SetAsyncRuntime(override bool) {
	r.asyncRuntime = override
}

func SolveAndUpdate2(request *Request, out output.Outputer) (_ *runtime.Runtime, rerr error) {
	defer rationalizeError(request.Auth, request.Proj, &rerr)

	if request.asyncRuntime {
		logging.Debug("Skipping runtime update due to async runtime")
		return nil, nil
	}

	target := target.NewProjectTarget(request.Proj, request.CustomCommitID, request.Trigger)
	rt, err := runtime.New(target, request.An, request.SvcModel, request.Auth, request.Cfg, out)
	if err != nil {
		return nil, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	if !bitflags.Has(request.Opts, OptOrderChanged) && !bitflags.Has(request.Opts, OptMinimalUI) && !rt.NeedsUpdate() {
		out.Notice(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return rt, nil
	}

	if rt.NeedsUpdate() && !bitflags.Has(request.Opts, OptMinimalUI) {
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
