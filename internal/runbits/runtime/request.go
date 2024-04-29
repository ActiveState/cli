package runtime

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Request struct {
	auth           *authentication.Auth
	analytics      analytics.Dispatcher
	project        *project.Project
	namespace      *model.Namespace
	customCommitID *strfmt.UUID
	trigger        target.Trigger
	svcModel       *model.SvcModel
	config         Configurable
	opts           Opts
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
		auth:           auth,
		analytics:      an,
		project:        proj,
		customCommitID: customCommitID,
		trigger:        trigger,
		svcModel:       svcm,
		config:         cfg,
		opts:           opts,
		asyncRuntime:   cfg.GetBool(constants.AsyncRuntimeConfig),
	}
}

func (r *Request) AsyncRuntime() bool {
	return r.asyncRuntime
}

func (r *Request) OverrideAsyncRuntime(override bool) {
	r.asyncRuntime = override
}

func (r *Request) SetNamespace(ns *model.Namespace) {
	r.namespace = ns
}
