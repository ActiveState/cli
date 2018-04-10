package hook

import (
	"fmt"

	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/commands"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/hooks"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/ActiveState/ActiveState-CLI/state/hook/add"
	"github.com/ActiveState/ActiveState-CLI/state/hook/remove"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "hook",
	Description: "hook_description",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "filter",
			Shorthand:   "",
			Description: "hook_filter_flag_usage",
			Type:        commands.TypeString,
			StringVar:   &flags.Filter,
		},
	},
}

// Flags for hook command
var flags struct {
	Filter string
}

func init() {
	Command.Append(add.Command)
	Command.Append(remove.Command)
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

// Execute List configured hooks
// If no hook trigger name given, lists all
// Otherwise shows configured hooks for given hook trigger
func Execute(cmd *cobra.Command, args []string) {
	var T = locale.T

	names := getFilters(cmd)
	project := projectfile.Get()

	hashmap, err := hooks.HashHooksFiltered(project.Hooks, names)
	if err != nil {
		failures.Handle(err, T("err_hook_cannot_list"))
	}

	print.Info(T("hook_listing_hooks"))
	print.Line()

	rows := [][]interface{}{}
	for k, hook := range hashmap {
		rows = append(rows, []interface{}{k, hook.Name, hook.Value})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{T("hook_header_id"), T("hook_header_hook"), T("hook_header_command")})
	t.SetAlign("left")

	hookmap, err := hooks.FilterHooks(hooknames)
	if err != nil {
		failures.Handle(err, locale.T("hook_hooks_not_available"))
		return
	}

	printOutput(hookmap)

	//logging.Debug("Execute:\n    hooknames: %v\n    hookmap: %v", hooknames, hookmap)
	print.Line(t.Render("simple"))
}
