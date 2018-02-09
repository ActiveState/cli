package hooks

import (
	"fmt"

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
		return "", fmt.Errorf("Cannot hash hook struct: %v", err)
	}
	return fmt.Sprintf("%X", hash), nil
}

// MapHooks creates a map of hooknames to associated commands
func MapHooks(hooks []projectfile.Hook) (map[string][]Hashedhook, error) {
	logging.Debug("mapHooks")
	hookmap := make(map[string][]Hashedhook)
	for _, hook := range hooks {
		hash, err := HashHookStruct(hook)
		// If we can't hash, something is really wrong so fail gracefully
		if err != nil {
			return nil, fmt.Errorf("Failed to map hooks: %v", err)
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
		return nil, fmt.Errorf("Failed to filter hooks: %v", err)
	}

	hookmap, err := MapHooks(config.Hooks)
	if err != nil {
		return nil, fmt.Errorf("Failed to filter hooks: %v", err)
	}
	// If no filters just return the whole thing
	if len(hooknames) == 0 {
		return hookmap, nil
	}

	var newmap = make(map[string][]Hashedhook)
	for _, val := range hooknames {
		if hookmap[val] != nil {
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
