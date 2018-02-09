package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"

	"github.com/ActiveState/ActiveState-CLI/internal/constraints"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/mitchellh/hashstructure"
)

// Hashedhook to easily associate a Hook struct to a hash of itself
type Hashedhook struct {
	Hook projectfile.Hook
	Hash string
}

// HashHookStruct takes a projectfile.Hook, hashes the struct and returns the hash as a string
func HashHookStruct(hook projectfile.Hook) (string, error) {
	hash, err := hashstructure.Hash(hook, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", hash), nil
}

// GetEffectiveHooks returns effective hooks by the given name, meaning only the ones that apply to the current runtime environment
func GetEffectiveHooks(hookName string, project *projectfile.Project) ([]*projectfile.Hook, error) {
	hooks := []*projectfile.Hook{}

	for _, hook := range project.Hooks {
		if hook.Name == hookName {
			if !constraints.IsConstrained(hook.Constraints, project) {
				hooks = append(hooks, &hook)
			}
		}
	}

	return hooks, nil
}

// RunHook runs effective hooks by the given name, meaning only the ones that apply to the current runtime environment
func RunHook(hookName string, project *projectfile.Project) error {
	hooks, err := GetEffectiveHooks(hookName, project)
	if err != nil {
		return err
	}

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

// MapHooks creates a map of hooknames to associated commands
func MapHooks(hooks []projectfile.Hook) (map[string][]Hashedhook, error) {
	logging.Debug("mapHooks")
	hookmap := make(map[string][]Hashedhook)
	for _, hook := range hooks {
		hash, err := HashHookStruct(hook)
		// If we can't hash, something is really wrong so fail gracefully
		if err != nil {
			return nil, err
		}
		newhook := Hashedhook{hook, hash}
		hookmap[hook.Name] = append(hookmap[hook.Name], newhook)
	}
	return hookmap, nil
}

// FilterHooks includes only hooks requested in a hookmap
func FilterHooks(hooknames []string) (map[string][]Hashedhook, error) {
	logging.Debug("filterHooks")
	config, err := projectfile.Get()
	if err != nil {
		return nil, err
	}

	hookmap, err := MapHooks(config.Hooks)
	if err != nil {
		return nil, err
	}
	// If no filters just return the whole thing
	if len(hooknames) == 0 {
		return hookmap, nil
	}

	var newmap = make(map[string][]Hashedhook)
	for _, val := range hooknames {
		if hookmap[val] != nil {
			// TODO: if !constraints.IsConstrained(hookmap[val].Hook.Constraints, config)
			newmap[val] = hookmap[val]
		}
	}

	//Empty array means nothing found in dict
	if len(newmap) == 0 {
		logging.Debug("No configured hooks for `%v`", hooknames)
		return nil, nil
	}
	return newmap, nil
	// logging.Debug("%v", hooknames)
}
