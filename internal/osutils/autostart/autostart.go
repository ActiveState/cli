package autostart

import (
	"github.com/ActiveState/cli/internal/errs"
)

type AppName string

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
	ConfigKey      string
	SetConfig      bool
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
	if a.options.SetConfig {
		if err := a.cfg.Set(a.options.ConfigKey, false); err != nil {
			return errs.Wrap(err, "Could not set config key")
		}
	}
	return a.enable()
}

func (a *app) EnableFirstTime() error {
	if a.options.SetConfig {
		if a.cfg.IsSet(a.options.ConfigKey) {
			return nil
		}
		if err := a.cfg.Set(a.options.ConfigKey, false); err != nil {
			return errs.Wrap(err, "Could not set config key")
		}
	}
	return a.enable()
}

func (a *app) Disable() error {
	if a.options.SetConfig {
		if err := a.cfg.Set(a.options.ConfigKey, true); err != nil {
			return errs.Wrap(err, "Could nto set config key")
		}
	}
	return a.disable()
}
