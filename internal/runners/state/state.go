package state

import (
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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

// Run state logic
func (s *State) Run(usageFunc func() error) error {
	return execute(s.opts, usageFunc, s.cfg, s.svcMdl, s.out)
}

type versionData struct {
	License    string `json:"license"`
	Version    string `json:"version"`
	Branch     string `json:"branch"`
	Revision   string `json:"revision"`
	Date       string `json:"date"`
	BuiltViaCI bool   `json:"builtViaCI"`
}

func execute(opts *Options, usageFunc func() error, cfg *config.Instance, svcModel *model.SvcModel, out output.Outputer) error {
	logging.Debug("Execute")
	defer profile.Measure("runners:state:execute", time.Now())

	if opts.Version {
		checker.RunUpdateNotifier(svcModel, out)
		vd := versionData{
			constants.LibraryLicense,
			constants.Version,
			constants.BranchName,
			constants.RevisionHash,
			constants.Date,
			constants.OnCI == "true",
		}
		out.Print(
			output.NewFormatter(vd).
				WithFormat(output.PlainFormatName, locale.T("version_info", vd)),
		)
		return nil
	}
	return usageFunc()
}
