package config

import (
	"os"
	"sort"
	"strings"

	"github.com/spf13/cast"
)

// EnvVarPrefix is prepended to a config key to derive its canonical environment-variable override.
const EnvVarPrefix = "ACTIVESTATE_CONFIG_"

var envVarReplacer = strings.NewReplacer(".", "_", "-", "_")

// CanonicalEnvVarName returns the environment variable that overrides the given config key. Every
// registered config option can be overridden this way, e.g. "update.info.endpoint" maps to
// "ACTIVESTATE_CONFIG_UPDATE_INFO_ENDPOINT".
func CanonicalEnvVarName(key string) string {
	return EnvVarPrefix + strings.ToUpper(envVarReplacer.Replace(key))
}

type Type int

const (
	String Type = iota
	Int
	Bool
	Enum
)

// Event is run when a user tries to set or get a config value via `state config`
type Event func(value interface{}) (interface{}, error)

var EmptyEvent = func(value interface{}) (interface{}, error) {
	if enum, ok := value.(*Enums); ok {
		// In case this config option is not set, return its default value instead
		// of the Enums struct itself.
		return enum.Default, nil
	}
	return value, nil
}

// Option defines what a config value's name and type should be, along with any get/set events
type Option struct {
	Name    string
	Type    Type
	Default interface{}
	// EnvAliases are additional (legacy/bespoke) environment variables that override this option, in
	// addition to its canonical CanonicalEnvVarName. They are kept for backwards compatibility with
	// env vars that predate the canonical ACTIVESTATE_CONFIG_* scheme (e.g. ACTIVESTATE_API_HOST).
	EnvAliases   []string
	GetEvent     Event
	SetEvent     Event
	isRegistered bool
	isHidden     bool
}

type Registry map[string]Option

var registry = make(Registry)

type Enums struct {
	Options []string
	Default string
}

func NewEnum(options []string, default_ string) *Enums {
	return &Enums{options, default_}
}

// GetOption returns a config option, regardless of whether or not it has been registered.
// Use KnownOption to determine if the returned option has been previously registered.
func GetOption(key string) Option {
	rule, ok := registry[key]
	if !ok {
		return Option{key, String, "", nil, EmptyEvent, EmptyEvent, false, false}
	}
	return rule
}

// Registers a config option without get/set events.
func RegisterOption(key string, t Type, defaultValue interface{}) {
	registerOption(key, t, defaultValue, nil, EmptyEvent, EmptyEvent, false)
}

// Registers a hidden config option without get/set events.
func RegisterHiddenOption(key string, t Type, defaultValue interface{}) {
	registerOption(key, t, defaultValue, nil, EmptyEvent, EmptyEvent, true)
}

// RegisterOptionWithEnv registers a config option along with one or more legacy/bespoke environment
// variables that override it, in addition to its canonical ACTIVESTATE_CONFIG_* variable.
func RegisterOptionWithEnv(key string, t Type, defaultValue interface{}, envAliases ...string) {
	registerOption(key, t, defaultValue, envAliases, EmptyEvent, EmptyEvent, false)
}

// Registers a config option with get/set events.
func RegisterOptionWithEvents(key string, t Type, defaultValue interface{}, get, set Event) {
	registerOption(key, t, defaultValue, nil, get, set, false)
}

func registerOption(key string, t Type, defaultValue interface{}, envAliases []string, get, set Event, hidden bool) {
	registry[key] = Option{key, t, defaultValue, envAliases, get, set, true, hidden}
}

// EnvVarNames returns every environment variable that can override this option: its canonical
// ACTIVESTATE_CONFIG_* variable first, followed by any legacy aliases.
func EnvVarNames(opt Option) []string {
	names := make([]string, 0, len(opt.EnvAliases)+1)
	names = append(names, CanonicalEnvVarName(opt.Name))
	names = append(names, opt.EnvAliases...)
	return names
}

// EnvOverride returns the effective override value for the option when one of its environment
// variables is currently set to a non-empty value, coerced to the option's type. The second return
// value is the name of the variable in effect; the bool reports whether an override applies.
func EnvOverride(opt Option) (interface{}, string, bool) {
	for _, name := range EnvVarNames(opt) {
		if v, ok := os.LookupEnv(name); ok && v != "" {
			return coerceToType(opt.Type, v), name, true
		}
	}
	return nil, "", false
}

// coerceToType converts a raw environment-variable string to the option's configured type so that
// callers receive the same Go type they would get from a stored value.
func coerceToType(t Type, raw string) interface{} {
	switch t {
	case Bool:
		return cast.ToBool(raw)
	case Int:
		return cast.ToInt(raw)
	default: // String, Enum
		return raw
	}
}

func KnownOption(rule Option) bool {
	return rule.isRegistered
}

func GetDefault(opt Option) interface{} {
	if enum, ok := opt.Default.(*Enums); ok {
		return enum.Default
	}
	return opt.Default
}

// Registered returns all registered options, excluding hidden ones
func Registered() []Option {
	var opts []Option
	for _, opt := range registry {
		if opt.isHidden {
			continue
		}
		opts = append(opts, opt)
	}
	sort.SliceStable(opts, func(i, j int) bool {
		return opts[i].Name < opts[j].Name
	})
	return opts
}
