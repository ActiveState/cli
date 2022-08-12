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
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func main() {
	var exitCode int

	var cfg *config.Instance
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}

		if cfg != nil {
			if err := cfg.Close(); err != nil {
				multilog.Error("Failed to close config after exiting systray: %v", err)
			}
		}

		if err := events.WaitForEvents(1*time.Second, rollbar.Wait, authentication.LegacyClose, logging.Close); err != nil {
			logging.Warning("Failed to wait events")
		}
		os.Exit(exitCode)
	}()

	var err error
	cfg, err = config.New()
	if err != nil {
		multilog.Critical("Could not initialize config: %v", errs.JoinMessage(err))
		fmt.Fprintf(os.Stderr, "Could not load config, if this problem persists please reinstall the State Tool. Error: %s\n", errs.JoinMessage(err))
		exitCode = 1
		return
	}
	rollbar.SetupRollbar(constants.StateTrayRollbarToken) // We're using the state tray project cause it's closely related
	rollbar.SetConfig(cfg)

	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	err = run(cfg)
	if err != nil {
		exitCode = 1
		multilog.Critical("Update Dialog Failure: " + errs.Join(err, ": ").Error())
		fmt.Fprintln(os.Stderr, errs.Join(err, ": ").Error())
		return
	}
}

func run(cfg *config.Instance) (rerr error) {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not initialize config")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	a := NewApp(cfg)
	if err := a.Start(); err != nil {
		return errs.Wrap(err, "Could not start application")
	}
	return nil
}
