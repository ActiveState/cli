package autostart

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
)

const ConfigKeyDisabled = "SystrayAutoStartDisabled"

type AppName string

func (a AppName) String() string {
	return string(a)
}

const (
	Tray    AppName = constants.TrayAppName
	Service         = constants.SvcAppName
)

type App struct {
	Name    string
	Exec    string
	cfg     Configurable
	options options
}

type Configurable interface {
	Set(string, interface{}) error
	IsSet(string) bool
}

func New(name AppName, exec string, cfg Configurable) *App {
	return &App{
		Name:    name.String(),
		Exec:    exec,
		cfg:     cfg,
		options: data[name],
	}
}

func (a *App) Enable() error {
	if err := a.cfg.Set(ConfigKeyDisabled, false); err != nil {
		return errs.Wrap(err, "ConfigKeyDisabled=false failed")
	}
	return a.enable()
}

func (a *App) EnableFirstTime() error {
	if a.cfg.IsSet(ConfigKeyDisabled) {
		return nil
	}
	if err := a.cfg.Set(ConfigKeyDisabled, false); err != nil {
		return errs.Wrap(err, "ConfigKeyDisabled=false failed")
	}
	return a.enable()
}

func (a *App) Disable() error {
	if err := a.cfg.Set(ConfigKeyDisabled, true); err != nil {
		return errs.Wrap(err, "ConfigKeyDisabled=true failed")
	}
	return a.disable()
}
