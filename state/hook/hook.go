package hook

import (
	"crypto/md5"
	"fmt"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/ActiveState/ActiveState-CLI/state/hook/add"
	"github.com/ActiveState/ActiveState-CLI/state/hook/remove"
	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
)

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "hook",
	Description: "hook_description",
	Run:         Execute,
}

// Flags for hook command
var Flags struct {
	Filter string
}

func init() {
	Command.Append(add.Command)
	Command.Append(remove.Command)
	// TODO make this work properly
	// It shows `--filter` in the --help information but when you run
	// `state hook --filter blah` it fails claiming it doesn't know what `--filter`
	// is
	//Command.GetCobraCmd().LocalFlags().StringVar(&Flags.Filter, "filter", "", locale.T("hook_filter_flag_usage"))
}

// Creates a map of hooknames to associated commands
func mapHooks(hooks []projectfile.Hook) map[string][]string {
	logging.Debug("mapHooks")
	hookmap := make(map[string][]string)
	for _, hook := range hooks {
		hash := fmt.Sprintf("%X", md5.Sum([]byte(hook.Name+hook.Value)))
		hookmap[hook.Name] = append(hookmap[hook.Name], hook.Value+" "+hash)
	}
	return hookmap
}

func filterHooks(hooknames []string) map[string][]string {
	logging.Debug("filterHooks")
	config, err := projectfile.Get()
	if err != nil {
		return nil
	}

	hookmap := mapHooks(config.Hooks)
	if len(hooknames) == 0 {
		return hookmap
	}

	var newmap = make(map[string][]string)
	for i := range hooknames {
		newmap[hooknames[i]] = hookmap[hooknames[i]]
	}
	//Empty array means nothing found in dict
	if len(newmap) == 0 {
		logging.Debug(locale.T("No configured hooks for `{{.Hooknames}}`", map[string]interface{}{"Hooknames": hooknames}))
		return nil
	}
	return newmap
	// logging.Debug("%v", hooknames)
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
func printOutput(hookmap map[string][]string) {
	logging.Debug("printOutput")
	for k, cmds := range hookmap {
		fmt.Printf(k + "\n")
		for idx := range cmds {
			fmt.Printf("\t" + cmds[idx] + "\n")
		}
	}
}

// Execute List configured hooks
// If no hook trigger name given, lists all
// Otherwise shows configured hooks for given hook trigger
func Execute(cmd *cobra.Command, args []string) {
	hooknames := getFilters(cmd)

	hookmap := filterHooks(hooknames)

	printOutput(hookmap)

	//logging.Debug("Execute:\n    hooknames: %v\n    hookmap: %v", hooknames, hookmap)
}
