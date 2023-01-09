package app

import (
	"github.com/ActiveState/cli/internal/constants"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

func init() {
	configMediator.RegisterOption(constants.AutostartSvcConfigKey, configMediator.Bool, configMediator.EmptyEvent, configMediator.EmptyEvent)
}

type App struct {
	Name    string
	Exec    string
	Args    []string
	options Options
}

type Options struct {
	LaunchFileName string
	IconFileName   string
	IconFileSource string
	GenericName    string
	Comment        string
	Keywords       string
	MacLabel       string // macOS autostart plist Label
	MacInteractive bool   // macOS autostart plist Interactive ProcessType
}

type Configurable interface {
	Set(string, interface{}) error
	IsSet(string) bool
}

func New(name string, exec string, args []string, opts Options, cfg Configurable) (*App, error) {
	return &App{
		Name: name,
		Exec: exec,
		Args: args,
	}, nil
}

func (a *App) Install() error {
	return a.install()
}

func (a *App) Uninstall() error {
	return a.uninstall()
}

func (a *App) EnableAutostart() error {
	return a.enableAutostart()
}

func (a *App) DisableAutostart() error {
	return a.disableAutostart()
}

func (a *App) IsAutostartEnabled() (bool, error) {
	return a.isAutostartEnabled()
}

func (a *App) AutostartInstallPath() (string, error) {
	return a.autostartInstallPath()
}
