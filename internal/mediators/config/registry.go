package config

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
	Name         string
	Type         Type
	Default      interface{}
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
		return Option{key, String, "", EmptyEvent, EmptyEvent, false, false}
	}
	return rule
}

// Registers a config option without get/set events.
func RegisterOption(key string, t Type, defaultValue interface{}) {
	registerOption(key, t, defaultValue, EmptyEvent, EmptyEvent, false)
}

// Registers a hidden config option without get/set events.
func RegisterHiddenOption(key string, t Type, defaultValue interface{}) {
	registerOption(key, t, defaultValue, EmptyEvent, EmptyEvent, true)
}

// Registers a config option with get/set events.
func RegisterOptionWithEvents(key string, t Type, defaultValue interface{}, get, set Event) {
	registerOption(key, t, defaultValue, get, set, false)
}

func registerOption(key string, t Type, defaultValue interface{}, get, set Event, hidden bool) {
	registry[key] = Option{key, t, defaultValue, get, set, true, hidden}
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

// AllRegistered returns all registered options, excluding hidden ones
func AllRegistered() []Option {
	var opts []Option
	for _, opt := range registry {
		if opt.isHidden {
			continue
		}
		opts = append(opts, opt)
	}
	return opts
}
