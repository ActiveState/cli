package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/exithandler"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/rollbar/rollbar-go"
)

func main() {
	var err error
	defer exithandler.Handle(err, func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, errs.JoinMessage(err))
		}
		events.WaitForEvents(1*time.Second, rollbar.Close)
		if err != nil {
			os.Exit(1)
		}
	})

	verbose := os.Getenv("VERBOSE") != ""
	logging.CurrentHandler().SetVerbose(verbose)
	logging.SetupRollbar(constants.StateToolRollbarToken)

	err = run()
}

func run() error {
	// handle state export config --filter=dir (install scripts call this function to write the install-source file)
	if len(os.Args) == 4 && os.Args[1] == "export" && os.Args[2] == "config" && os.Args[3] == "--filter=dir" {
		cfg, err := config.New()
		if err != nil {
			return errs.Wrap(err, "Failed to read configuration.")
		}
		fmt.Println(cfg.ConfigPath())
		return nil
	}

	if len(os.Args) < 1 || os.Args[1] != "_prepare" {
		fmt.Println("Sorry! This is a transitional tool that should have been replaced during the last update.   If you see this message, something must have gone wrong.  Re-trying to update now...")
	}

	up, err := updater.DefaultChecker.GetUpdateInfo("", "")
	if err != nil {
		return errs.Wrap(err, "Failed to check for latest update.")
	}

	err = up.InstallBlocking()
	if err != nil {
		return errs.Wrap(err, "Failed to install mult-file update.")
	}

	return nil
}
