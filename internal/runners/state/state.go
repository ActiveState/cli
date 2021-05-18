package state

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
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
	opts *Options
	out  output.Outputer
	cfg  *config.Instance
}

type primeable interface {
	primer.Outputer
	primer.Configurer
}

func New(opts *Options, prime primeable) *State {
	return &State{
		opts: opts,
		out:  prime.Output(),
		cfg:  prime.Config(),
	}
}

// Run state logic
func (s *State) Run(usageFunc func() error) error {
	return execute(s.opts, usageFunc, s.cfg, s.out)
}

type versionData struct {
	License  string `json:"license"`
	Version  string `json:"version"`
	Branch   string `json:"branch"`
	Revision string `json:"revision"`
	Date     string `json:"date"`
}

func execute(opts *Options, usageFunc func() error, cfg *config.Instance, out output.Outputer) error {
	logging.Debug("Execute")

	if opts.Version {
		checker.RunUpdateNotifier(cfg, out)
		vd := versionData{
			constants.LibraryLicense,
			constants.Version,
			constants.BranchName,
			constants.RevisionHash,
			constants.Date,
		}
		out.Print(
			output.NewFormatter(vd).
				WithFormat(output.PlainFormatName, locale.T("version_info", vd)),
		)
		return nil
	}
	return usageFunc()
}
