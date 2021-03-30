package main

import (
	"os"

	"github.com/ActiveState/cli/cmd/state-tray/internal/open"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/getlantern/systray"
	"github.com/gobuffalo/packr"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	err := run()
	if err != nil {
		logging.Error("Systray encountered an error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	logging.CurrentHandler().SetVerbose(true)

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

	mQuit := systray.AddMenuItem(locale.Tl("tray_exit", "Exit"), "")

	for {
		select {
		case <-mAbout.ClickedCh:
			logging.Debug("About event")
			err := open.Prompt("state --version")
			if err != nil {
				logging.Error("Could not open command prompt, got error: %v", err)
			}
		case <-mDoc.ClickedCh:
			logging.Debug("Documentation event")
			// Not implemented
		case <-mLearn.ClickedCh:
			logging.Debug("Learn event")
			// Not implemented
		case <-mSupport.ClickedCh:
			logging.Debug("Support event")
			// Not implemented
		case <-mAccount.ClickedCh:
			logging.Debug("Account event")
			// Not implemented
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
