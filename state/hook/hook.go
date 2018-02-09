package hook

import (
	"fmt"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/ActiveState/cobra"
	"github.com/mitchellh/hashstructure"
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

// Hashedhook to easily associate a Hook struct to a hash of itself
type Hashedhook struct {
	Hook projectfile.Hook
	Hash string
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

// HashHookStruct takes a projectfile.Hook, hashes the struct and returns the hash as a string
func HashHookStruct(hook projectfile.Hook) string {
	hash, err := hashstructure.Hash(hook, nil)
	if err != nil {
		logging.Error("Cannot hash hook struct: %v", err)
		return ""
	}
	return fmt.Sprintf("%X", hash)
}

// MapHooks creates a map of hooknames to associated commands
func MapHooks(hooks []projectfile.Hook) map[string][]Hashedhook {
	logging.Debug("mapHooks")
	hookmap := make(map[string][]Hashedhook)
	for _, hook := range hooks {
		hash := HashHookStruct(hook)
		// If we can't hash, something is really wrong so fail gracefully
		//
		if hash == "" {
			print.Warning(locale.T("hook_cannot_hash_warning"))
			return nil
		}
		newhook := Hashedhook{hook, hash}
		hookmap[hook.Name] = append(hookmap[hook.Name], newhook)
	}
	return hookmap
}

// FilterHooks includes only hooks requested in a hookmap
func FilterHooks(hooknames []string) map[string][]Hashedhook {
	logging.Debug("filterHooks")
	config, err := projectfile.Get()
	if err != nil {
		return nil
	}

	hookmap := MapHooks(config.Hooks)
	if len(hooknames) == 0 {
		return hookmap
	}

	var newmap = make(map[string][]Hashedhook)
	for i := range hooknames {
		newmap[hooknames[i]] = hookmap[hooknames[i]]
	}

	//Empty array means nothing found in dict
	if len(newmap) == 0 {
		logging.Debug("No configured hooks for `%v`", hooknames)
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
func printOutput(hookmap map[string][]Hashedhook) {
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

	hookmap := FilterHooks(hooknames)

	printOutput(hookmap)

	//logging.Debug("Execute:\n    hooknames: %v\n    hookmap: %v", hooknames, hookmap)
}
