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
	IconFileName    string
	IconFileSource  string
	GenericName     string
	Comment         string
	Keywords        string
	IsGUIApp        bool
	MacLabel        string // macOS autostart plist Label
	MacInteractive  bool   // macOS autostart plist Interactive ProcessType
	MacHideDockIcon bool
}

type Configurable interface {
	Set(string, interface{}) error
	IsSet(string) bool
}

func New(name string, exec string, args []string, opts Options, cfg Configurable) (*App, error) {
	return &App{
		Name:    name,
		Exec:    exec,
		Args:    args,
		options: opts,
	}, nil
}

func (a *App) Install() error {
	return a.install()
}

func (a *App) Uninstall() error {
	return a.uninstall()
}
