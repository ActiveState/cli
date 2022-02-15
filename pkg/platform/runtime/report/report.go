package report

import (
	"context"
	"os"
	"strconv"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Targeter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Dir() string
	Headless() bool
	Trigger() target.Trigger

	// OnlyUseCache communicates that this target should only use cached runtime information (ie. don't check for updates)
	OnlyUseCache() bool
}

type Report struct {
	d          analytics.Dispatcher
	svcm       *model.SvcModel
	targeter   Targeter
	instanceID string
	used       bool
}

func New(d analytics.Dispatcher, svcm *model.SvcModel, t Targeter, instanceID string) *Report {
	instanceid.ID()
	return &Report{
		d:          d,
		svcm:       svcm,
		targeter:   t,
		instanceID: instanceID,
	}
}

func (r *Report) RuntimeStart() (runtimeCached func()) {
	r.d.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeStart, &dimensions.Values{
		Trigger:          p.StrP(r.targeter.Trigger().String()),
		Headless:         p.StrP(strconv.FormatBool(r.targeter.Headless())),
		CommitID:         p.StrP(r.targeter.CommitUUID().String()),
		ProjectNameSpace: p.StrP(project.NewNamespace(r.targeter.Owner(), r.targeter.Name(), r.targeter.CommitUUID().String()).String()),
		InstanceID:       p.StrP(r.instanceID),
	})

	runtimeCached = func() {
		r.d.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeCache)
	}

	return runtimeCached
}

func (r *Report) RuntimeConclusion(err error, label string) {
	action := anaConsts.ActRuntimeFailure
	if locale.IsInputError(err) {
		action = anaConsts.ActRuntimeUserFailure
	}
	r.d.EventWithLabel(anaConsts.CatRuntime, action, label)
}

func (r *Report) RuntimeSuccess() {
	r.d.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeSuccess)
}

func (r *Report) RuntimeUse() {
	if r.targeter.Trigger().IndicatesUsage() {
		r.recordUsage()
	}
}

func (r *Report) RuntimeBuild() {
	r.d.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeBuild)
	ns := project.Namespaced{
		Owner:   r.targeter.Owner(),
		Project: r.targeter.Name(),
	}
	r.d.EventWithLabel(anaConsts.CatRuntime, anaConsts.ActBuildProject, ns.String())
}

func (r *Report) RuntimeDownload() {
	r.d.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeDownload)
}

func (r *Report) recordUsage() {
	if !r.used {
		defer func() { r.used = true }()

		dims := &dimensions.Values{
			Trigger:          p.StrP(r.targeter.Trigger().String()),
			Headless:         p.StrP(strconv.FormatBool(r.targeter.Headless())),
			CommitID:         p.StrP(r.targeter.CommitUUID().String()),
			ProjectNameSpace: p.StrP(project.NewNamespace(r.targeter.Owner(), r.targeter.Name(), r.targeter.CommitUUID().String()).String()),
			InstanceID:       p.StrP(r.instanceID),
		}
		dimsJson, err := dims.Marshal()
		if err != nil {
			logging.Critical("Could not marshal dimensions for runtime-usage: %s", errs.JoinMessage(err))
		}

		if r.svcm != nil {
			// TODO: handle error if needed
			r.svcm.RecordRuntimeUsage(context.Background(), os.Getpid(), osutils.Executable(), dimsJson) //nolint
		}
	}
}
