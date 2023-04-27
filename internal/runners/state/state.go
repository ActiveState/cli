package state

import (
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Options struct {
	Locale      string
	Version     bool
	ConfirmExit bool
}

func NewOptions() *Options {
	return &Options{}
}

type State struct {
	opts   *Options
	out    output.Outputer
	cfg    *config.Instance
	svcMdl *model.SvcModel
}

type primeable interface {
	primer.Outputer
	primer.Configurer
	primer.SvcModeler
}

func New(opts *Options, prime primeable) *State {
	return &State{
		opts:   opts,
		out:    prime.Output(),
		cfg:    prime.Config(),
		svcMdl: prime.SvcModel(),
	}
}

type versionOutput struct {
	message string
	installation.VersionData
}

func (o *versionOutput) MarshalOutput(format output.Format) interface{} {
	return o.message
}

func (o *versionOutput) MarshalStructured(format output.Format) interface{} {
	return o.VersionData
}

func (s *State) Run(usageFunc func() error) error {
	defer profile.Measure("runners:state:run", time.Now())

	if s.opts.Version {
		checker.RunUpdateNotifier(s.svcMdl, s.out)
		vd := installation.VersionData{
			constants.LibraryLicense,
			constants.Version,
			constants.BranchName,
			constants.RevisionHash,
			constants.Date,
			constants.OnCI == "true",
		}
		s.out.Print(&versionOutput{
			locale.T("version_info", vd),
			vd,
		})
		return nil
	}
	return usageFunc()
}
