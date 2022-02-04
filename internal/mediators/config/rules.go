package config

type Type int

const (
	String Type = iota
	Int
	Bool
)

// Event is run when a user tries to set or get a config value via `state config`
type Event func(value interface{}) (interface{}, error)

var EmptyEvent = func(value interface{}) (interface{}, error) { return nil, nil }

// Rule defines what type the config value should be along with any get/set events
type Rule struct {
	Type     Type
	GetEvent Event
	SetEvent Event
}

var defaultRule = Rule{String, EmptyEvent, EmptyEvent}

type Rules map[string]Rule

var rules Rules

func GetRule(key string) Rule {
	rule, ok := rules[key]
	if !ok {
		return defaultRule
	}
	return rule
}

func NewRule(key string, t Type, get Event, set Event) {
	rules[key] = Rule{t, get, set}
}

func SetRule(key string, rule Rule) {
	rules[key] = rule
}

func init() {
	rules = make(Rules)
}
