package main

import (
	"context"
	"os"

	"github.com/ActiveState/cli/cmd/state-tray/internal/menu"
	"github.com/ActiveState/cli/cmd/state-tray/internal/open"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/getlantern/systray"
	"github.com/gobuffalo/packr"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	err := run()
	if err != nil {
		logging.Error("Systray encountered an error: %v", errs.Join(err, ": "))
		os.Exit(1)
	}
}

func run() error {
	logging.CurrentHandler().SetVerbose(true)

	config, err := config.Get()
	if err != nil {
		return errs.Wrap(err, "Could not get new config instance")
	}

	model, err := model.NewSvcModel(context.Background(), config)
	if err != nil {
		return errs.Wrap(err, "Could not create new service model")
	}

	box := packr.NewBox("assets")
	systray.SetIcon(box.Bytes("icon.ico"))
	systray.SetTooltip(locale.Tl("tray_tooltip", "ActiveState State Tool"))

	mAbout := systray.AddMenuItem(
		locale.Tl("tray_about_title", "About State Tool"),
		locale.Tl("tray_about_tooltip", "Information about the State Tool"),
	)

	systray.AddSeparator()

	mDoc := systray.AddMenuItem(
		locale.Tl("tray_documentation_title", "Documentation"),
		locale.Tl("tray_documentation_tooltip", "Open State Tool Docs"),
	)

	mPlatform := systray.AddMenuItem(locale.Tl("tray_platform_title", "ActiveState Platform"), "")
	mLearn := mPlatform.AddSubMenuItem(
		locale.Tl("tray_learn_title", "Learn"),
		locale.Tl("tray_learn_tooltip", "Open ActiveState blog"),
	)
	mSupport := mPlatform.AddSubMenuItem(
		locale.Tl("tray_support_title", "Support"),
		locale.Tl("tray_support_tooltip", "Open support page"),
	)
	mAccount := mPlatform.AddSubMenuItem(
		locale.Tl("tray_account_title", "Account"),
		locale.Tl("tray_account_tooltip", "Open your account page"),
	)

	systray.AddSeparator()

	mProjects := systray.AddMenuItem(locale.Tl("tray_projects_title", "Local Projects"), "")
	mReload := mProjects.AddSubMenuItem("Reload", "Reload the local projects listing")
	localProjectsUpdater := menu.NewLocalProjectsUpdater(mProjects)

	localProjects, err := model.LocalProjects()
	if err != nil {
		logging.Error("Could not get local projects listing: %v", err)
	}
	localProjectsUpdater.Update(localProjects)

	systray.AddSeparator()

	mQuit := systray.AddMenuItem(locale.Tl("tray_exit", "Exit"), "")

	for {
		select {
		case <-mAbout.ClickedCh:
			logging.Debug("About event")
			err = open.Prompt("state --version")
			if err != nil {
				logging.Error("Could not open command prompt: %v", err)
			}
		case <-mDoc.ClickedCh:
			logging.Debug("Documentation event")
			err = open.Browser("https://docs.activestate.com/platform/state/")
			if err != nil {
				logging.Error("Could not open documentation url: %v", err)
			}
		case <-mLearn.ClickedCh:
			logging.Debug("Learn event")
			err = open.Browser("https://www.activestate.com/blog/")
			if err != nil {
				logging.Error("Could not open blog url: %v", err)
			}
		case <-mSupport.ClickedCh:
			logging.Debug("Support event")
			err = open.Browser("https://www.activestate.com/support/")
			if err != nil {
				logging.Error("Could not open support url: %v", err)
			}
		case <-mAccount.ClickedCh:
			logging.Debug("Account event")
			// Not implemented
		case <-mReload.ClickedCh:
			logging.Debug("Projects event")
			localProjects, err = model.LocalProjects()
			if err != nil {
				logging.Error("Could not get local projects listing: %v", err)
			}
			localProjectsUpdater.Update(localProjects)
		case <-mQuit.ClickedCh:
			logging.Debug("Quit event")
			systray.Quit()
			return nil
		}
	}
}

func onExit() {
	// Not implemented
}
