package main

import (
	"github.com/ActiveState/cli/cmd/state-tray/assets"
	"github.com/ActiveState/cli/cmd/state-tray/internal/open"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/getlantern/systray"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(assets.Icon)
	systray.SetTitle("ActiveState State Tool")
	systray.SetTooltip("ActiveState State Tool")

	mAbout := systray.AddMenuItem("About State Tool", "Information about the State Tool")

	systray.AddSeparator()

	mDoc := systray.AddMenuItem("Documentation", "Open State Tool Docs")

	mPlatform := systray.AddMenuItem("ActiveState Platform", "")
	mLearn := mPlatform.AddSubMenuItem("Learn", "ActiveState Blog")
	mSupport := mPlatform.AddSubMenuItem("Support", "Open support page")
	mAccount := mPlatform.AddSubMenuItem("Account", "Open your account page")

	systray.AddSeparator()

	// TODO: Populate the local projects entries at application startup
	// mProjects := systray.AddMenuItem("Local Projects", "")
	// systray.AddSeparator()

	mQuit := systray.AddMenuItem("Exit", "")

	for {
		select {
		case <-mAbout.ClickedCh:
			err := open.Prompt("state --version")
			if err != nil {
				logging.Error("Could not start command, got error: %v", err)
			}
		case <-mDoc.ClickedCh:
			// Do stuff
		case <-mLearn.ClickedCh:
			// Do stuff
		case <-mSupport.ClickedCh:
			// Do stuff
		case <-mAccount.ClickedCh:
			// Do stuff
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func onExit() {
	// clean up here
}
