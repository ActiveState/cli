package hooks

import (
	"os"
	"os/exec"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	funk "github.com/thoas/go-funk"

	"github.com/ActiveState/ActiveState-CLI/internal/constraints"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
)

// HashedHook to easily associate a Hook struct to a hash of itself
type HashedHook struct {
	Hook projectfile.Hook
	Hash string
}

// GetEffectiveHooks returns effective hooks by the given name, meaning only the ones that apply to the current runtime environment
func GetEffectiveHooks(hookName string, project *projectfile.Project) []*projectfile.Hook {
	hooks := []*projectfile.Hook{}

	for _, hook := range project.Hooks {
		if hook.Name == hookName {
			if !constraints.IsConstrained(hook.Constraints, project) {
				hooks = append(hooks, &hook)
			}
		}
	}

	return hooks
}

// RunHook runs effective hooks by the given name, meaning only the ones that apply to the current runtime environment
func RunHook(hookName string, project *projectfile.Project) error {
	hooks := GetEffectiveHooks(hookName, project)

	if len(hooks) == 0 {
		return nil
	}

	// This is an exception to the rule, since RunHook can be called from many different controllers and since we
	// want to communicate the command being ran we have a print statement here, this is not ideal and should otherwise
	// be avoided
	print.Info(locale.T("info_running_hook", map[string]interface{}{"Name": hookName}))

	for _, hook := range hooks {
		// Todo: Find a library to properly split command strings
		args := strings.Split(hook.Value, " ")

		print.Info("> " + hook.Value)

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

// HashHooks returns a map of all the hooks with the keys being a hash of that hook
func HashHooks(hooks []projectfile.Hook) (map[string]projectfile.Hook, error) {
	hashedHooks := make(map[string]projectfile.Hook)
	for _, hook := range hooks {
		hash, err := hook.Hash()
		// If we can't hash, something is really wrong so fail gracefully
		if err != nil {
			return nil, err
		}
		hashedHooks[hash] = hook
	}
	return hashedHooks, nil
}

// HashHooksFiltered is identical to HashHooks except that it takes a slice of names to be used as a filter
func HashHooksFiltered(hooks []projectfile.Hook, hookNames []string) (map[string]projectfile.Hook, error) {
	hashedHooks, err := HashHooks(hooks)
	if err != nil {
		return nil, err
	}
	if len(hookNames) == 0 {
		return hashedHooks, err
	}

	hashedHooksFiltered := make(map[string]projectfile.Hook)
	for hash, hook := range hashedHooks {
		if funk.Contains(hookNames, hook.Name) {
			hashedHooksFiltered[hash] = hook
		}
	}

	return hashedHooksFiltered, nil
}

// PromptOptions returns an array of strings that can be consumed by the survey library we use,
// the second return argument contains a map that connects each item to a hash
func PromptOptions(project *projectfile.Project, filter string) ([]string, map[string]string, error) {
	optionsMap := make(map[string]string)
	options := []string{}

	filters := []string{}
	if filter != "" {
		filters = append(filters, filter)
	}

	hashedHooks, err := HashHooksFiltered(project.Hooks, filters)
	if err != nil {
		return options, optionsMap, err
	}

	if len(hashedHooks) == 0 {
		return options, optionsMap, failures.User.New(locale.T("err_hook_cannot_find"))
	}

	for hash, hook := range hashedHooks {
		command := strings.Replace(hook.Value, "\n", " ", -1)
		if len(command) > 50 {
			command = command[0:50] + ".."
		}

		constraints := []string{}
		if hook.Constraints.Environment != "" {
			constraints = append(constraints, hook.Constraints.Environment)
		}
		if hook.Constraints.Platform != "" {
			constraints = append(constraints, hook.Constraints.Platform)
		}

		var constraintString string
		if len(constraints) > 0 {
			constraintString = strings.Join(constraints, ", ") + ", "
		}

		value := locale.T("prompt_hook_option", map[string]interface{}{
			"Hash":        hash,
			"Hook":        hook,
			"Command":     command,
			"Constraints": constraintString,
		})
		options = append(options, value)
		optionsMap[value] = hash
	}

	return options, optionsMap, nil
}
