package runtime

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Request struct {
	Auth           *authentication.Auth
	Out            output.Outputer
	Analytics      analytics.Dispatcher
	Project        *project.Project
	Namespace      *model.Namespace
	CustomCommitID *strfmt.UUID
	Trigger        target.Trigger
	SvcModel       *model.SvcModel
	Config         Configurable
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
) *Request {

	return &Request{
		Auth:           auth,
		Analytics:      an,
		Project:        proj,
		CustomCommitID: customCommitID,
		Trigger:        trigger,
		SvcModel:       svcm,
		Config:         cfg,
		Opts:           opts,
		asyncRuntime:   cfg.GetBool(constants.AsyncRuntimeConfig),
	}
}

func (r *Request) Async() bool {
	return r.asyncRuntime
}

func (r *Request) SetAsyncRuntime(override bool) {
	r.asyncRuntime = override
}

func (r *Request) SetNamespace(ns *model.Namespace) {
	r.Namespace = ns
}
