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
	"github.com/ActiveState/cli/internal/exithandler"
	"github.com/ActiveState/cli/internal/logging"
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

	logging.SetupRollbar(constants.StateTrayRollbarToken) // We're using the state tray project cause it's closely related

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	err = run()
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
