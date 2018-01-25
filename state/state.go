package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/ActiveState-CLI/internal/config" // MUST be first!
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"

	"github.com/ActiveState/ActiveState-CLI/state/install"

	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
)

var T = locale.T

// Flags hold the flag values passed through the command line
var Flags struct {
	Locale string
}

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "state",
	Description: "state_description",
	Run:         Execute,

	UsageTemplate: "usage_tpl",
}

func init() {
	logging.Debug("init")

	cC := Command.GetCobraCmd()
	cC.PersistentFlags().StringVarP(&Flags.Locale, "locale", "l", "", T("flag_state_locale_description"))

	Command.Append(install.Command)
}

func main() {
	logging.Debug("main")

	// This actually runs the command
	err := Command.Execute()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Write our config to file
	config.Save()
}

// Execute the `state` command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
}
