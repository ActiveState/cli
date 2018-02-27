package remove

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

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
func printOutput(hookmap map[string][]hookhelper.Hashedhook) {
	var T = locale.T

	print.Line()
	print.Info(T("hook_listing_hooks"))
	print.Line()

	rows := [][]interface{}{}
	for k, cmds := range hookmap {
		for idx := range cmds {
			rows = append(rows, []interface{}{idx + 1, cmds[idx].Hash, k, cmds[idx].Hook.Value})
		}
	}
	t := gotabulate.Create(rows)
	t.SetHeaders([]string{T("hook_header_index"), T("hook_header_id"), T("hook_header_hook"), T("hook_header_command")})
	t.SetAlign("left")
	print.Line(t.Render("simple"))
	print.Line(locale.T("hook_remove_multiple_hooks"))
}

//  Cycle through the configured hooks, hash then remove hook if matches, save, exit
func removeByHash(identifier string, project *projectfile.Project) bool {
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

// Index is the human readable idx, ie. first pos is 1, not 0
func removeByIndex(idx int, hooks []hookhelper.Hashedhook, project *projectfile.Project) {
	// Check if it's out of range
	// otherwise carry on with next line
	hookLen := len(hooks)
	if hookLen < idx || 0 > idx {
		err := failures.User.New(locale.T("hook_remove_index_out_of_range"))
		failures.Handle(err, "")
		return
	}
	removeByHash(hooks[idx-1].Hash, project)
}

// Execute the hook remove command
// Adds a statement to be run on the given hook
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute `hook remove`")

	project, err := projectfile.Get()
	if err != nil {
		err = failures.User.New(err.Error())
		failures.Handle(err, locale.T("hook_remove_cannot_remove", Args))
		return
	}

	if removeByHash(Args.Identifier, project) {
		return
	}

	filteredMappedHooks, err := hookhelper.FilterHooks([]string{Args.Identifier})
	if err != nil {
		failures.Handle(err, locale.T("hook_remove_cannot_remove", Args))
		return
	}

	fileredHooks := filteredMappedHooks[Args.Identifier]
	numOfHooksFound := len(fileredHooks)
	if numOfHooksFound < 1 { // Hook not found
		print.Info(locale.T("hook_remove_hook_not_found", map[string]interface{}{"Hookname": Args.Identifier}))
	} else if numOfHooksFound == 1 { // Hook found! Remove it.
		removeByName(Args.Identifier, project)
	} else if numOfHooksFound >= 2 { // Multiple hooks found.  List them.
		printOutput(filteredMappedHooks)
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		index, _ := reader.ReadString('\n')
		idx, err := strconv.Atoi(strings.TrimSpace(index))
		if err == nil {
			removeByIndex(idx, fileredHooks, project)
		} else {
			failures.Handle(err, "Couldn't remove indexed hook")
		}
	}
}
