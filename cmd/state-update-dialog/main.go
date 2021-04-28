package main

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/rollbar/rollbar-go"
)

func main() {
	var exit int
	logging.SetupRollbar(constants.StateTrayRollbarToken) // We're using the state tray project cause it's closely related
	defer func() {
		events.WaitForEvents(1*time.Second, rollbar.Close)
		os.Exit(exit)
	}()

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	err := run()
	if err != nil {
		exit = 1
		logging.Error("Update Dialog Failure: " + errs.Join(err, ": ").Error())
		fmt.Fprintln(os.Stderr, errs.Join(err, ": ").Error())
	}
}

func run() error {
	a := NewApp()
	if err := a.Start(); err != nil {
		return errs.Wrap(err, "Could not start application")
	}
	return nil
}
