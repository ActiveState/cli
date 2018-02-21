package remove

import (
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/commands"
	hookhelper "github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/hooks"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/bndr/gotabulate"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/cobra"
)

// Command remove, sub command of hook
var Command = &commands.Command{
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
func removebyHash(identifier string, project *projectfile.Project) bool {
	hooks := project.Hooks
	var removed = false
	for i, hook := range hooks {
		hash, err := hookhelper.HashHookStruct(hook)
		if identifier == hash {
			hooks := append(hooks[:i], hooks[i+1:]...)
			project.Hooks = hooks
			removed = true
			break
		} else if err != nil {
			logging.Warning("Failed to remove hook '%v': %v", identifier, err)
			print.Warning(locale.T("hook_remove_cannot_remove", map[string]interface{}{"Hookname": identifier, "Error": err}))
		}
	}
	project.Save()
	return removed
}

// Print what we ended up with
func printOutput(hookmap map[string][]hookhelper.Hashedhook) {
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

func removeByName(identifier string, project *projectfile.Project) {
	hooks := project.Hooks
	for i, hook := range hooks {
		if identifier == hook.Name {
			hooks := append(hooks[:i], hooks[i+1:]...)
			project.Hooks = hooks
			break
		}
	}
	project.Save()
}

// Execute the hook remove command
// Adds a statement to be run on the given hook
func Execute(cmd *cobra.Command, args []string) {
	identifier := args[0]
	project, err := projectfile.Get()
	if err != nil {
		logging.Error("%v", err)
		print.Warning(locale.T("hook_remove_cannot_remove", map[string]interface{}{"Hookname": identifier, "Error": err}))
		return
	}

	if removebyHash(identifier, project) {
		return
	}

	mappedHooks, err := hookhelper.FilterHooks([]string{identifier})
	if err != nil {
		print.Warning(locale.T("hook_remove_cannot_remove", map[string]interface{}{"Hookname": identifier, "Error": err}))
		logging.Error("%v", err)
		return
	}

	numOfHooksFound := len(mappedHooks[identifier])
	if numOfHooksFound == 1 {
		removeByName(identifier, project)

	} else if numOfHooksFound >= 2 {
		printOutput(mappedHooks)
	}
	logging.Debug("Execute `hook remove`")
}
