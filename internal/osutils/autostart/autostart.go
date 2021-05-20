package autostart

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
)

const ConfigKeyDisabled = "SystrayAutoStartDisabled"

type App struct {
	Name string
	Exec string
	cfg  *config.Instance
}

func New(name, exec string, cfg *config.Instance) *App {
	return &App{
		Name: name,
		Exec: exec,
		cfg:  cfg,
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
