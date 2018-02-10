package remove

import (
	helper "github.com/ActiveState/ActiveState-CLI/internal/helpers"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/bndr/gotabulate"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/cobra"
)

// Command remove, sub command of hook
var Command = &structures.Command{
	Name:        "remove",
	Description: "remove_description",
	Run:         Execute,
}

func init() {
	Command.GetCobraCmd().Args = validateArgs
}

func validateArgs(cmd *cobra.Command, args []string) error {
	// TODO lists of known hooks, warn if hook passed isn't supported
	// err := cobra.OnlyValidArgs(cmd, args)
	// if err != nil {
	// 	return err
	// }
	err := cobra.ExactArgs(1)(cmd, args)
	if err != nil {
		return err
	}
	return nil
}

//  Cycle through the configured hooks, hash then remove hook if matches, save, exit
func removebyHash(identifier string, config *projectfile.Project) bool {
	hooks := config.Hooks
	var removed = false
	for i, hook := range hooks {
		hash, err := helper.HashHookStruct(hook)
		if identifier == hash {
			hooks := append(hooks[:i], hooks[i+1:]...)
			config.Hooks = hooks
			removed = true
			break
		} else if err != nil {
			logging.Warning("Failed to remove hook '%v': %v", identifier, err)
		}
	}
	projectfile.Write(projectfile.GetProjectFilePath(), config)
	return removed
}

// Print what we ended up with
func printOutput(hookmap map[string][]helper.Hashedhook) {
	var T = locale.T

	print.Info(T("hook_listing_hooks"))
	print.Line()

	rows := [][]interface{}{}
	for k, cmds := range hookmap {
		for idx := range cmds {
			rows = append(rows, []interface{}{cmds[idx].Hash, k, cmds[idx].Hook.Value})
		}
	}
	t := gotabulate.Create(rows)
	t.SetHeaders([]string{T("hook_header_id"), T("hook_header_hook"), T("hook_header_command")})
	t.SetAlign("left")
	print.Line(t.Render("simple"))
	print.Line(locale.T("hook_remove_multiple_hooks"))
}

func removeByName(identifier string, config *projectfile.Project) {
	hooks := config.Hooks
	for i, hook := range hooks {
		if identifier == hook.Name {
			hooks := append(hooks[:i], hooks[i+1:]...)
			config.Hooks = hooks
			break
		}
	}
	projectfile.Write(projectfile.GetProjectFilePath(), config)
}

// Execute the hook remove command
// Adds a statement to be run on the given hook
func Execute(cmd *cobra.Command, args []string) {
	identifier := args[0]
	config, err := projectfile.Get()
	if err != nil {
		logging.Error("%v", err)
		return
	}

	if removebyHash(identifier, config) {
		return
	}

	mappedHooks, err := helper.FilterHooks([]string{identifier})
	logging.Warning("This is my mappedhooks: %V", mappedHooks)
	if err != nil {
		logging.Error("%v", err)
		return
	}

	numOfHooksFound := len(mappedHooks[identifier])
	logging.Warning("The length of this badboy is: %v", numOfHooksFound)
	if numOfHooksFound == 1 {
		removeByName(identifier, config)
	} else if numOfHooksFound >= 2 {
		printOutput(mappedHooks)
	}

	logging.Debug("Execute `hook remove`")
}
