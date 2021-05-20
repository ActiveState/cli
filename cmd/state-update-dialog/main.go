package main

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/rollbar/rollbar-go"
)

func main() {
	var exitCode int
	defer func() {
		if runbits.HandlePanics() {
			exitCode = 1
		}
		events.WaitForEvents(1*time.Second, rollbar.Close)
		os.Exit(exitCode)
	}()

	logging.SetupRollbar(constants.StateTrayRollbarToken) // We're using the state tray project cause it's closely related

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	err := run()
	if err != nil {
		exitCode = 1
		logging.Error("Update Dialog Failure: " + errs.Join(err, ": ").Error())
		fmt.Fprintln(os.Stderr, errs.Join(err, ": ").Error())
		return
	}
}

func run() error {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}

	a := NewApp(cfg)
	if err := a.Start(); err != nil {
		return errs.Wrap(err, "Could not start application")
	}
	return nil
}
