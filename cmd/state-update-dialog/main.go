package main

import (
	_ "embed"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/rollbar/rollbar-go"
)

func main() {
	var exitCode int
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}
		if err := events.WaitForEvents(1*time.Second, rollbar.Close, authentication.LegacyClose); err != nil {
			logging.Warning("Failed to wait for rollbar to close")
		}
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

func run() (rerr error) {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	machineid.Configure(cfg)
	machineid.SetErrorLogger(logging.Error)

	a := NewApp(cfg)
	if err := a.Start(); err != nil {
		return errs.Wrap(err, "Could not start application")
	}
	return nil
}
