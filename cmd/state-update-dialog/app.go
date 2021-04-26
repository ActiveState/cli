package main

import (
	_ "embed"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/wailsapp/wails"
)

//go:embed frontend/main.html
var html string

//go:embed frontend/main.js
var js string

//go:embed frontend/main.css
var css string

type App struct {
	wails *wails.App
}

func NewApp() *App {
	a := &App{}
	a.wails = wails.CreateApp(&wails.AppConfig{
		Width:            1024,
		Height:           768,
		Title:            "ActiveState Desktop - Update Available",
		HTML:             html,
		JS:               js,
		CSS:              css,
		Colour:           "#FFF", // Wails uses this to detect dark mode
		Resizable:        false,
		DisableInspector: false,
	})
	return a
}

func (a *App) CurrentVersion() string {
	return constants.Version
}

func (a *App) Start() error {
	// var err error
	// a.update, err = updater.DefaultChecker.Check()
	// if err != nil {
	//	return errs.Wrap(err, "Could not check for updates")
	// }
	update := updater.NewAvailableUpdate("2.0.0", "release", "darwin", "", "")
	if update == nil {
		return errs.New("No updates available")
	}
	a.wails.Bind(&Bindings{update})
	return a.wails.Run()
}
