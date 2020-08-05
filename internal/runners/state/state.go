package state

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
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
}

func New(opts *Options) *State {
	return &State{
		opts: opts,
	}
}

// Run state logic
func (s *State) Run(usageFunc func() error) error {
	return execute(s.opts, usageFunc)
}

func execute(opts *Options, usageFunc func() error) error {
	logging.Debug("Execute")

	if opts.Version {
		print.Info(locale.T("version_info", map[string]interface{}{
			"License":  constants.LibraryLicense,
			"Version":  constants.Version,
			"Branch":   constants.BranchName,
			"Revision": constants.RevisionHash,
			"Date":     constants.Date}))
		return nil
	}
	return usageFunc()
}
