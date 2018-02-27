package remove

import (
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/commands"
	hookhelper "github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/hooks"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/bndr/gotabulate"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/spf13/cobra"
)

// Args hold the arg values passed through the command line
var Args struct {
	Identifier string
}

// Command remove, sub command of hook
var Command = &commands.Command{
	Name:        "remove",
	Description: "remove_description",
	Run:         Execute,

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_hook_remove_identifier",
			Description: "arg_hook_remove_identifier_description",
			Variable:    &Args.Identifier,
			Required:    true,
		},
	},
}

// Print what we ended up with
func printOutput(hookmap map[string][]hookhelper.HashedHook) {
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

// Execute the hook remove command
// Adds a statement to be run on the given hook
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute `hook remove`")

	project, err := projectfile.Get()
	if err != nil {
		failures.Handle(err, locale.T("hook_remove_cannot_remove", Args))
		return
	}

	if removebyHash(project) {
		return
	}

	hashedHooks, err := hookhelper.HashHooksFiltered(project.Hooks, []string{Args.Identifier})
	if err != nil {
		failures.Handle(err, locale.T("hook_remove_cannot_remove", Args))
		return
	}

	numOfHooksFound := len(hashedHooks)
	if numOfHooksFound == 1 {
		removeByName(project)

	} else if numOfHooksFound >= 2 {
		//removeByPrompt(project) // under construction
	}
}

//  Cycle through the configured hooks, hash then remove hook if matches, save, exit
func removebyHash(project *projectfile.Project) bool {
	hooks := project.Hooks
	var removed = false
	for i, hook := range hooks {
		hash, err := hook.Hash()
		if Args.Identifier == hash {
			hooks := append(hooks[:i], hooks[i+1:]...)
			project.Hooks = hooks
			removed = true
			break
		} else if err != nil {
			logging.Warning("Failed to remove hook '%v': %v", Args.Identifier, err)
			print.Warning(locale.T("hook_remove_cannot_remove", map[string]interface{}{"Hookname": Args.Identifier, "Error": err}))
		}
	}
	project.Save()
	return removed
}

func removeByName(project *projectfile.Project) {
	hooks := project.Hooks
	for i, hook := range hooks {
		if Args.Identifier == hook.Name {
			hooks := append(hooks[:i], hooks[i+1:]...)
			project.Hooks = hooks
			break
		}
	}
	project.Save()
}

// func removeByPrompt(project *projectfile.Project) {
// 	hooks := hooks.MapHooks(project.Hooks)
// 	if hooks[Args.Identifier] == nil {
// 		return
// 	}

// 	options := []string{}

// 	for i, hashedHook := range hooks[Args.Identifier] {
// 		hash, err := hook.Hash()
// 		if Args.Identifier == hash {
// 			options = append(options)
// 			break
// 		} else if err != nil {
// 			logging.Warning("Failed to remove hook '%v': %v", Args.Identifier, err)
// 			print.Warning(locale.T("hook_remove_cannot_remove", map[string]interface{}{"Hookname": Args.Identifier, "Error": err}))
// 		}
// 	}
// 	project.Save()

// 	var qs = []*survey.Question{
// 		{
// 			Name: "color",
// 			Prompt: &survey.Select{
// 				Message: locale.T("prompt_choose_hook"),
// 				Options: []string{"red", "blue", "green"},
// 			},
// 		},
// 	}
// }
