package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/client/sync"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/errors"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/updater"
)

type Params struct {
	channel        string
	force          bool
	version        string
	nonInteractive bool
}

func newParams() *Params {
	return &Params{}
}

var filenameRe = regexp.MustCompile(`(?P<name>[^/\\]+?)_(?P<webclientId>[^/\\_.]+)(\.(?P<ext>[^.]+))?$`)

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

	// Store sessionToken to config
	webclientId := "remote_" + constants.RemoteInstallerVersion
	if matches := filenameRe.FindStringSubmatch(os.Args[0]); matches != nil {
		if index := filenameRe.SubexpIndex("webclientId"); index != -1 {
			webclientId = matches[index]
		} else {
			multilog.Error("Invalid subexpression ID for webclient ID")
		}
	}
	err = cfg.Set(anaConst.CfgSessionToken, webclientId)
	if err != nil {
		logging.Error("Unable to set session token: " + errs.JoinMessage(err))
	}

	an = sync.New(anaConst.SrcStateRemoteInstaller, cfg, nil, out)

	// Set up prompter
	prompter := prompt.New(true, an)

	params := newParams()
	cmd := captain.NewCommand(
		"state-installer",
		"",
		"Installs or updates the State Tool",
		primer.New(out, cfg, an),
		[]*captain.Flag{ // The naming of these flags is slightly inconsistent due to backwards compatibility requirements
			{
				Name:        "channel",
				Description: "Defaults to 'release'.  Specify an alternative channel to install from (eg. beta)",
				Value:       &params.channel,
			},
			{
				Shorthand: "b", // backwards compatibility
				Hidden:    true,
				Value:     &params.channel,
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
			{
				Name:      "non-interactive",
				Shorthand: "n",
				Hidden:    true,
				Value:     &params.nonInteractive,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return execute(out, prompter, cfg, an, args, params)
		},
	)

	err = cmd.Execute(os.Args[1:])
	if err != nil {
		errors.ReportError(err, cmd, an)
		exitCode, err = errors.ParseUserFacing(err)
		if err != nil {
			out.Error(err)
		}
		return
	}
}

func execute(out output.Outputer, prompt prompt.Prompter, cfg *config.Instance, an analytics.Dispatcher, args []string, params *Params) error {
	msg := locale.Tr("tos_disclaimer", constants.TermsOfServiceURLLatest)
	msg += locale.Tr("tos_disclaimer_prompt", constants.TermsOfServiceURLLatest)
	cont, err := prompt.Confirm(locale.Tr("install_remote_title"), msg, ptr.To(true))
	if err != nil {
		return errs.Wrap(err, "Could not prompt for confirmation")
	}

	if !cont {
		return locale.NewInputError("install_cancel", "Installation cancelled")
	}

	channel := params.channel
	if channel == "" {
		channel = constants.ReleaseChannel
	}

	// Fetch payload
	checker := updater.NewDefaultChecker(cfg, an)
	checker.InvocationSource = updater.InvocationSourceInstall // Installing from a remote source is only ever encountered via the install flow
	availableUpdate, err := checker.CheckFor(channel, params.version)
	if err != nil {
		return errs.Wrap(err, "Could not retrieve install package information")
	} else if availableUpdate == nil {
		return locale.NewError("remote_install_no_available_update", "Could not find installer to download. This could be due to networking issues or temporary maintenance. Please try again later.")
	}

	version := availableUpdate.Version
	if params.channel != "" {
		version = fmt.Sprintf("%s (%s)", version, channel)
	}

	update := updater.NewUpdateInstaller(an, availableUpdate)
	out.Fprint(os.Stdout, locale.Tl("remote_install_downloading", "• Downloading State Tool version [NOTICE]{{.V0}}[/RESET]... ", version))
	tmpDir, err := update.DownloadAndUnpack()
	if err != nil {
		out.Print(locale.Tl("remote_install_status_fail", "[ERROR]x Failed[/RESET]"))
		return errs.Wrap(err, "Could not download and unpack")
	}
	out.Print(locale.Tl("remote_install_status_done", "[SUCCESS]✔ Done[/RESET]"))

	out.Print(locale.Tl("remote_install_status_running", "• Running Installer..."))
	if params.nonInteractive {
		args = append(args, "-n") // forward to installer
	}
	env := []string{
		constants.InstallerNoSubshell + "=true",
	}
	_, cmd, err := osutils.ExecuteAndPipeStd(filepath.Join(tmpDir, constants.StateInstallerCmd+osutils.ExeExtension), args, env)
	if err != nil {
		if cmd != nil && cmd.ProcessState.Sys().(syscall.WaitStatus).Exited() {
			// The issue happened while running the command itself, meaning the responsibility for conveying the error
			// is on the command, rather than us.
			return errs.Silence(errs.Wrap(err, "Installer failed"))
		}
		return errs.Wrap(err, "Could not run installer")
	}

	out.Print(locale.Tl("remote_install_exit_prompt", "Press ENTER to exit."))
	fmt.Scanln(ptr.To("")) // Wait for input from user

	return nil
}
