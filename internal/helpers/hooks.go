package hooks

import (
	"fmt"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/mitchellh/hashstructure"
)

// Hashedhook to easily associate a Hook struct to a hash of itself
type Hashedhook struct {
	Hook projectfile.Hook
	Hash string
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
