package config

import "github.com/ActiveState/cli/internal/constants"

type event func() error

var emptyEvent = func() error { return nil }

type configType int

const (
	String configType = iota
	Int
	Bool
)

type configRule struct {
	allowedType configType
	getEvent    event
	setEvent    event
}

type configRules map[Key]configRule

func (c configRules) Get(key Key) configRule {
	rule, ok := c[key]
	if !ok {
		return configRule{String, emptyEvent, emptyEvent}
	}
	return rule
}

var rules = configRules{
	constants.SvcConfigPid:  {Int, emptyEvent, emptyEvent},
	constants.SvcConfigPort: {Int, emptyEvent, emptyEvent},
}
