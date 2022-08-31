package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/client/sync"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/cmdlets/errors"
)

type Params struct {
	branch  string
	force   bool
	version string
}

func newParams() *Params {
	return &Params{}
}

func main() {
	var exitCode int

	var an analytics.Dispatcher

	var cfg *config.Instance

	// Handle things like panics, exit codes and the closing of globals
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}

		if err := cfg.Close(); err != nil {
			logging.Error("Failed to close config: %w", err)
		}

		if err := events.WaitForEvents(5*time.Second, rollbar.Wait, an.Wait, logging.Close); err != nil {
			logging.Warning("state-remote-installer failed to wait for events: %v", err)
		}
		os.Exit(exitCode)
	}()

	// Set up verbose logging
	logging.CurrentHandler().SetVerbose(os.Getenv("VERBOSE") != "")
	// Set up rollbar reporting
	rollbar.SetupRollbar(constants.StateInstallerRollbarToken)

	// Allow starting the installer via a double click
	captain.DisableMousetrap()

	// Set up configuration handler
	cfg, err := config.New()
	if err != nil {
		logging.Error("Could not set up configuration handler: " + errs.JoinMessage(err))
		fmt.Fprintln(os.Stderr, err.Error())
		exitCode = 1
	}

	rollbar.SetConfig(cfg)

	// Set up output handler
	out, err := output.New("plain", &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     true,
		Interactive: false,
	})
	if err != nil {
		logging.Error("Could not set up output handler: " + errs.JoinMessage(err))
		fmt.Fprintln(os.Stderr, err.Error())
		exitCode = 1
		return
	}

	an = sync.New(cfg, nil)

	// Set up prompter
	prompter := prompt.New(true, an)

	params := newParams()
	cmd := captain.NewCommand(
		"state-installer",
		"",
		"Installs or updates the State Tool",
		primer.New(nil, out, nil, nil, nil, nil, cfg, nil, nil, an),
		[]*captain.Flag{ // The naming of these flags is slightly inconsistent due to backwards compatibility requirements
			{
				Name:        "channel",
				Description: "Defaults to 'release'.  Specify an alternative channel to install from (eg. beta)",
				Value:       &params.branch,
			},
			{
				Shorthand: "b", // backwards compatibility
				Hidden:    true,
				Value:     &params.branch,
			},
			{
				Name:        "version",
				Shorthand:   "v",
				Description: "The version of the State Tool to install",
				Value:       &params.version,
			},
			{
				Name:        "force",
				Shorthand:   "f",
				Description: "Force the installation, overwriting any version of the State Tool already installed",
				Value:       &params.force,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return execute(out, prompter, cfg, an, args, params)
		},
	)

	err = cmd.Execute(os.Args[1:])
	if err != nil {
		if locale.IsInputError(err) {
			logging.Error("Installer input error: " + errs.JoinMessage(err))
		} else {
			multilog.Critical("Installer error: " + errs.JoinMessage(err))
		}

		exitCode, err = errors.Unwrap(err)
		out.Error(err)
		return
	}
}

func execute(out output.Outputer, prompt prompt.Prompter, cfg *config.Instance, an analytics.Dispatcher, args []string, params *Params) error {
	msg := locale.Tr("tos_disclaimer", constants.TermsOfServiceURLLatest)
	msg += locale.Tr("tos_disclaimer_prompt", constants.TermsOfServiceURLLatest)
	cont, err := prompt.Confirm(locale.Tr("install_remote_title"), msg, p.BoolP(true))
	if err != nil {
		return errs.Wrap(err, "Could not prompt for confirmation")
	}

	if !cont {
		return locale.NewInputError("install_cancel", "Installation cancelled")
	}

	branch := params.branch
	if branch == "" {
		branch = constants.ReleaseBranch
	}

	// Fetch payload
	checker := updater.NewDefaultChecker(cfg)
	checker.InvocationSource = updater.InvocationSourceInstall // Installing from a remote source is only ever encountered via the install flow
	checker.VerifyVersion = false
	update, err := checker.CheckFor(branch, params.version)
	if err != nil {
		return errs.Wrap(err, "Could not retrieve install package information")
	}
	if update == nil {
		return errs.New("No update information could be found.")
	}

	version := update.Version
	if params.branch != "" {
		version = fmt.Sprintf("%s (%s)", version, branch)
	}

	out.Fprint(os.Stdout, fmt.Sprintf("• Downloading State Tool version [NOTICE]%s[/RESET]... ", version))
	tmpDir, err := update.DownloadAndUnpack()
	if err != nil {
		out.Print("[ERROR]x Failed[/RESET]")
		return errs.Wrap(err, "Could not download and unpack")
	}
	out.Print("[SUCCESS]✔ Done[/RESET]")

	env := []string{
		constants.InstallerNoSubshell + "=true",
	}
	_, _, err = exeutils.ExecuteAndPipeStd(filepath.Join(tmpDir, constants.StateInstallerCmd+exeutils.Extension), args, env)
	if err != nil {
		return errs.Wrap(err, "Could not run installer")
	}

	out.Print("Installation complete. Press enter to exit.")
	fmt.Scanln(p.StrP(""))

	return nil
}
