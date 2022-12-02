package autostart

import (
	"github.com/ActiveState/cli/internal/constants"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

type AppName string

func init() {
	configMediator.RegisterOption(constants.AutostartSvcConfigKey, configMediator.Bool, configMediator.EmptyEvent, configMediator.EmptyEvent)
}

func (a AppName) String() string {
	return string(a)
}

type app struct {
	Name    string
	Exec    string
	Args    []string
	cfg     Configurable
	options Options
}

type Options struct {
	LaunchFileName string
	IconFileName   string
	IconFileSource string
	GenericName    string
	Comment        string
	Keywords       string
	MacLabel       string // macOS plist Label
	MacInteractive bool   // macOS plist Interactive ProcessType
}

type Configurable interface {
	Set(string, interface{}) error
	IsSet(string) bool
}

func New(name AppName, exec string, args []string, options Options, cfg Configurable) (*app, error) {
	return &app{
		Name:    name.String(),
		Exec:    exec,
		Args:    args,
		cfg:     cfg,
		options: options,
	}, nil
}

func (a *app) Enable() error {
	return a.enable()
}

func (a *app) Disable() error {
	return a.disable()
}
