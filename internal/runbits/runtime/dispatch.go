package runtime

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Dispatcher struct {
	auth           *authentication.Auth
	out            output.Outputer
	an             analytics.Dispatcher
	proj           *project.Project
	customCommitID *strfmt.UUID
	trigger        target.Trigger
	svcModel       *model.SvcModel
	cfg            Configurable
	opts           Opts
	asyncRuntime   bool
}

func NewDispatcher(auth *authentication.Auth,
	an analytics.Dispatcher,
	proj *project.Project,
	customCommitID *strfmt.UUID,
	trigger target.Trigger,
	svcm *model.SvcModel,
	cfg Configurable,
	out output.Outputer,
	opts Opts,
) (_ *Dispatcher, rerr error) {
	defer rationalizeError(auth, proj, &rerr)

	if proj == nil {
		return nil, rationalize.ErrNoProject
	}

	if proj.IsHeadless() {
		return nil, rationalize.ErrHeadless
	}

	return &Dispatcher{
		auth:           auth,
		an:             an,
		proj:           proj,
		customCommitID: customCommitID,
		trigger:        trigger,
		svcModel:       svcm,
		cfg:            cfg,
		opts:           opts,
		out:            out,
		asyncRuntime:   cfg.GetBool(constants.AsyncRuntimeConfig),
	}, nil
}

// OverrideAsyncRuntime overrides the async runtime setting from the config.
func (d *Dispatcher) OverrideAsyncRuntime(override bool) {
	d.asyncRuntime = override
}

func (d *Dispatcher) SolveAndUpdate() (_ *runtime.Runtime, rerr error) {
	defer rationalizeError(d.auth, d.proj, &rerr)

	if d.asyncRuntime {
		logging.Debug("Skipping runtime update due to async runtime")
		return nil, nil
	}

	// TODO: Might need to make this private to not stack defers.
	// Also, it could return a wrapper around the runtime that can provide
	// additional inforamtion like whether the result actually contains a runtime
	// or if it wasn't deployed for whatever reason.
	// Could also return a typed error for specfic dispatch conditions.
	return SolveAndUpdate(d.auth, d.out, d.an, d.proj, d.customCommitID, d.trigger, d.svcModel, d.cfg, d.opts)
}
