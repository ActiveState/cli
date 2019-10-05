package state

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
)

type StateOptions struct {
	Locale  string
	Verbose bool
	Version bool
}

func NewStateOptions() *StateOptions {
	return &StateOptions{}
}

type StateRunner struct {
	opts *StateOptions
}

func NewStateRunner(opts *StateOptions) *StateRunner {
	sc := StateRunner{
		opts: opts,
	}

	return &sc
}

// Execute the `state` command
func (c *StateRunner) Execute(usageFunc func() error) error {
	return execute(c.opts, usageFunc)
}

func execute(opts *StateOptions, usageFunc func() error) error {
	logging.Debug("Execute")

	if opts.Version {
		print.Info(locale.T("version_info", map[string]interface{}{
			"Version":  constants.Version,
			"Branch":   constants.BranchName,
			"Revision": constants.RevisionHash,
			"Date":     constants.Date}))
		return nil
	}

	return usageFunc()
}

/*
func (c *StateRunner) onVerboseFlag() {
	if c.flagVerbose {
		logging.CurrentHandler().SetVerbose(true)
	}
}

RunE: func(cmd *captain.Command, args []string) error {
	if opts.Verbose {
		logging.CurrentH
	}
	sc := cmd.NewStateRunner(opts)
	sc.Execute(cmd.Usage)
}*/
