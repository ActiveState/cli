package autostart

import (
	"os"
	"path/filepath"

	as "github.com/emersion/go-autostart"

	"github.com/ActiveState/cli/internal/locale"
)

type App struct {
	app *as.App
}

func New() *App {
	return &App{
		&as.App{
			Name:        "activestate-desktop",
			DisplayName: locale.T("tray_app_name", "ActiveState Desktop"),
			Exec:        []string{filepath.Join(filepath.Dir(os.Args[0]), "state-tray")},
		},
	}
}

func (a *App) Enable() error {
	if a.IsEnabled() {
		return nil
	}
	return a.app.Enable()
}

func (a *App) Toggle() error {
	if a.IsEnabled() {
		return a.app.Disable()
	}
	return a.app.Enable()
}

func (a *App) Disable() error {
	if !a.IsEnabled() {
		return nil
	}
	return a.app.Disable()
}

func (a *App) IsEnabled() bool {
	return a.app.IsEnabled()
}
