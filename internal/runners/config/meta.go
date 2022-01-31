package config

import "github.com/ActiveState/cli/internal/constants"

type event func() error

type configType int

const (
	Int configType = iota
	Bool
)

type configMeta struct {
	Type     configType
	getEvent event
	setEvent event
}

var meta = map[string]configMeta{
	constants.SvcConfigPid:  {Int, nil, nil},
	constants.SvcConfigPort: {Int, nil, nil},
}
