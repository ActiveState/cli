package main

import (
	"os/exec"

	"github.com/ActiveState/cli/cmd/state-tray/assets"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/getlantern/systray"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(assets.Icon)
	systray.SetTitle("State Tool")
	systray.SetTooltip("State Tool")
	mAbout := systray.AddMenuItem("About State Tool", "Information about the State Tool")
	systray.AddSeparator()
	mDoc := systray.AddMenuItem("Documentation", "Open State Tool Docs")
	mPlatform := systray.AddMenuItem("ActiveState Platform", "")
	mLearn := mPlatform.AddSubMenuItem("Learn", "ActiveState Blog")
	mSupport := mPlatform.AddSubMenuItem("Support", "Open support page")
	mAccount := mPlatform.AddSubMenuItem("Account", "Open your account page")
	systray.AddSeparator()
	// mProjects := systray.AddMenuItem("Local Projects", "")
	// systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "")

	for {
		select {
		case <-mAbout.ClickedCh:
			cmd := exec.Command("cmd.exe", "/K", "start", "state --version")
			err := cmd.Run()
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
