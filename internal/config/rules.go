package config

type ConfigType int

const (
	String ConfigType = iota
	Int
	Bool
)

// Event is run when a user tries to set or get a config value via `state config`
type Event func(value interface{}) error

var emptyEvent = func(value interface{}) error { return nil }

// Rule defines what type the config value should be along with any get/set events
type Rule struct {
	Type     ConfigType
	GetEvent Event
	SetEvent Event
}

var defaultRule = Rule{String, emptyEvent, emptyEvent}

type Rules map[string]Rule

var rules = make(Rules)

func GetRule(key string) Rule {
	rule, ok := rules[key]
	if !ok {
		return defaultRule
	}
	return rule
}

func SetRule(key string, rule Rule) {
	rules[key] = rule
}
