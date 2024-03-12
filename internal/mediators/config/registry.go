package config

type Type int

const (
	String Type = iota
	Int
	Bool
)

// Event is run when a user tries to set or get a config value via `state config`
type Event func(value interface{}) (interface{}, error)

var EmptyEvent = func(value interface{}) (interface{}, error) { return value, nil }

// Option defines what a config value's name and type should be, along with any get/set events
type Option struct {
	Name         string
	Type         Type
	Default      interface{}
	GetEvent     Event
	SetEvent     Event
	isRegistered bool
}

type Registry map[string]Option

var registry = make(Registry)

// GetOption returns a config option, regardless of whether or not it has been registered.
// Use KnownOption to determine if the returned option has been previously registered.
func GetOption(key string) Option {
	rule, ok := registry[key]
	if !ok {
		return Option{key, String, "", EmptyEvent, EmptyEvent, false}
	}
	return rule
}

// Registers a config option.
// Name, Type, and Default are required fields.
func RegisterOption(option Option) {
	if option.GetEvent == nil {
		option.GetEvent = EmptyEvent
	}
	if option.SetEvent == nil {
		option.SetEvent = EmptyEvent
	}
	option.isRegistered = true
	registry[option.Name] = option
}

func KnownOption(rule Option) bool {
	return rule.isRegistered
}
