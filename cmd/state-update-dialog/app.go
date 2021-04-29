package main

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/ActiveState/cli/cmd/state-update-dialog/internal/lockedprj"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/httpreq"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/wailsapp/wails"
	"github.com/yuin/goldmark"
)

//go:embed frontend/main.html
var html string

//go:embed frontend/generated/main.js
var js string

//go:embed frontend/generated/main.css
var css string

type App struct {
	wails *wails.App
}

func NewApp() *App {
	a := &App{}
	a.wails = wails.CreateApp(&wails.AppConfig{
		Width:  600,
		Height: 400,
		Title:  "ActiveState Desktop - Update Available",
		HTML:   html,
		JS:     js,
		CSS:    css,
		Colour: "#FFF", // Wails uses this to detect dark mode
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

	bindings := &Bindings{}

	update := updater.NewAvailableUpdate("2.0.0", "release", "darwin", "", "")
	if update == nil {
		return errs.New("No updates available")
	}
	bindings.update = update

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not create config instance")
	}

	lockedProjects := lockedprj.LockedProjectMapping(cfg)
	bindings.lockedProjects = lockedProjects

	go func() {
		url := fmt.Sprintf("https://raw.githubusercontent.com/ActiveState/cli/%s/changelog.md", update.Channel)
		changelog, err := httpreq.New().Get(url)
		if err != nil {
			logging.Error(fmt.Sprintf("Could not retrieve changelog: %v", errs.Join(err, ": ")))
			return
		}

		var buf bytes.Buffer
		if err := goldmark.Convert(changelog, &buf); err != nil {
			logging.Error(fmt.Sprintf("Could not convert changelog to html: %v", errs.Join(err, ": ")))
			return
		}

		bindings.changelog = buf.String()
	}()

	a.wails.Bind(bindings)
	return a.wails.Run()
}
