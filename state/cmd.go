package main

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/spf13/cobra"
)

type StateCommand struct {
	meta      captain.Meta
	locale    captain.Locale
	flags     []*captain.Flag
	arguments []*captain.Argument
	options   []captain.Option

	flagLocale  string
	flagVerbose bool
	flagVersion bool
}

func NewStateCommand() *StateCommand {
	sc := StateCommand{}
	sc.meta = captain.Meta{
		Name: "state",
	}
	sc.locale = captain.Locale{
		Description:   locale.T("state_description"),
		UsageTemplate: "usage_tpl",
	}
	sc.flags = []*captain.Flag{
		{
			Name:        "locale",
			Shorthand:   "l",
			Description: locale.T("flag_state_locale_description"),
			Type:        captain.TypeString,
			Persist:     true,
			StringVar:   &sc.flagLocale,
		},
		{
			Name:        "verbose",
			Shorthand:   "v",
			Description: "flag_state_verbose_description",
			Type:        captain.TypeBool,
			Persist:     true,
			OnUse:       sc.onVerboseFlag,
			BoolVar:     &sc.flagVerbose,
		},
		{
			Name:        "version",
			Description: "flag_state_version_description",
			Type:        captain.TypeBool,
			BoolVar:     &sc.flagVersion,
		},
	}
	sc.arguments = []*captain.Argument{
		&captain.Argument{
			Name:        "arg_state_activate_namespace",
			Description: "arg_state_activate_namespace_description",
		},
	}
	sc.options = []captain.Option{}
	return &sc
}

func (c *StateCommand) Meta() captain.Meta {
	return c.meta
}

func (c *StateCommand) Locale() captain.Locale {
	return c.locale
}

func (c *StateCommand) Flags() []*captain.Flag {
	return c.flags
}

func (c *StateCommand) Arguments() []*captain.Argument {
	return c.arguments
}

func (c *StateCommand) Options() []captain.Option {
	return c.options
}

// Execute the `state` command
func (c *StateCommand) Execute(cmd *cobra.Command, args []string) error {
	logging.Debug("Execute")

	if c.flagVersion {
		print.Info(locale.T("version_info", map[string]interface{}{
			"Version":  constants.Version,
			"Branch":   constants.BranchName,
			"Revision": constants.RevisionHash,
			"Date":     constants.Date}))
		return nil
	}

	cmd.Usage()

	return nil
}

func (c *StateCommand) onVerboseFlag() {
	if c.flagVerbose {
		logging.CurrentHandler().SetVerbose(true)
	}
}
