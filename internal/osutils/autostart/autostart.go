package autostart

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
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

type app struct {
	Name    string
	Exec    string
	Args    []string
	cfg     Configurable
	options options
}

type Configurable interface {
	Set(string, interface{}) error
	IsSet(string) bool
}

func New(name AppName, exec string, args []string, cfg Configurable) (*app, error) {
	data, ok := data[name]
	if !ok {
		return nil, locale.NewError("err_autostart_unrecognized", "Unrecognized autostart app type")
	}

	return &app{
		Name:    name.String(),
		Exec:    exec,
		Args:    args,
		cfg:     cfg,
		options: data,
	}, nil
}

func (a *app) Enable() error {
	if a.Name == Tray.String() {
		if err := a.cfg.Set(ConfigKeyDisabled, false); err != nil {
			return errs.Wrap(err, "ConfigKeyDisabled=false failed")
		}
	}
	return a.enable()
}

func (a *app) EnableFirstTime() error {
	if a.Name == Tray.String() {
		if a.cfg.IsSet(ConfigKeyDisabled) {
			return nil
		}
		if err := a.cfg.Set(ConfigKeyDisabled, false); err != nil {
			return errs.Wrap(err, "ConfigKeyDisabled=false failed")
		}
	}
	return a.enable()
}

func (a *app) Disable() error {
	if a.Name == Tray.String() {
		if err := a.cfg.Set(ConfigKeyDisabled, true); err != nil {
			return errs.Wrap(err, "ConfigKeyDisabled=true failed")
		}
	}
	return a.disable()
}
