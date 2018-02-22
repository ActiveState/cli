package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/ActiveState-CLI/internal/config" // MUST be first!
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/commands"

	// commands
	"github.com/ActiveState/ActiveState-CLI/state/activate"
	"github.com/ActiveState/ActiveState-CLI/state/hook"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/spf13/cobra"
)

var exit = os.Exit

// T links to locale.T
var T = locale.T

// Flags hold the flag values passed through the command line
var Flags struct {
	Locale string
}

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "state",
	Description: "state_description",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "locale",
			Shorthand:   "l",
			Description: "flag_state_locale_description",
			Type:        commands.TypeString,
			Persist:     true,
			StringVar:   &Flags.Locale,
		},
	},

	UsageTemplate: "usage_tpl",
}

func init() {
	logging.Debug("init")

	Command.Append(activate.Command)
	Command.Append(hook.Command)
}

func main() {
	logging.Debug("main")

	// This actually runs the command
	err := Command.Execute()

	if err != nil {
		fmt.Println(err)
		exit(1)
		return
	}

	// Write our config to file
	config.Save()
}

// Execute the `state` command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
}
