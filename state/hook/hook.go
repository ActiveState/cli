package hook

import (
	"fmt"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/cobra"
)

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "hook",
	Description: "hook_description",
	Run:         Execute,
}

// Flags for hook command
var flags struct {
	Filter string
}

func init() {
	//Command.Append(add.Command)
	//Command.Append(remove.Command)
	// TODO make this work properly
	// It shows `--filter` in the --help information but when you run
	// `state hook --filter blah` it fails claiming it doesn't know what `--filter`
	// is
	//Command.GetCobraCmd().LocalFlags().StringVar(&flags.Filter, "filter", "", locale.T("hook_filter_flag_usage"))
}

func getFilters(cmd *cobra.Command) []string {
	logging.Debug("getFilters")
	// TODO: we should support comma seperate string of hooknames
	// OR
	// TODO support multiple --filter Flags
	// Using list of hooknames so that's easier later
	flags := cmd.LocalFlags().Lookup("filter")
	var hooknames []string
	if flags != nil && fmt.Sprintf("%v", flags.Value) != "" {
		hooknames = append(hooknames, fmt.Sprintf("%v", flags.Value))
	}
	// Alt. meth of retrieving flags
	// if Flags.Filter != "" {
	// 	hooknames = append(hooknames, Flags.Filter)
	// }
	return hooknames
}

// Print what we ended up with
func printOutput(hookmap map[string][]hooks.Hashedhook) {
	logging.Debug("printOutput")
	for k, cmds := range hookmap {
		print.Info(k + "\n")
		for idx := range cmds {
			print.Info("\t%v\n", cmds[idx])
		}
	}
}

// Execute List configured hooks
// If no hook trigger name given, lists all
// Otherwise shows configured hooks for given hook trigger
func Execute(cmd *cobra.Command, args []string) {
	hooknames := getFilters(cmd)

	hookmap := hooks.FilterHooks(hooknames)

	printOutput(hookmap)

	//logging.Debug("Execute:\n    hooknames: %v\n    hookmap: %v", hooknames, hookmap)
}
