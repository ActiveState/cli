package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/rollbar/rollbar-go"
)

type Params struct {
	fromDeferred    bool
	sourcePath      string
	path            string
	updateTag       string
	branch          string
	command         string
	version         string
	activate        *project.Namespaced
	activateDefault *project.Namespaced
}

func newParams() *Params {
	return &Params{activate: &project.Namespaced{}, activateDefault: &project.Namespaced{}}
}

func main() {
	var exitCode int

	// Handle things like panics, exit codes and the closing of globals
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}
		if err := events.WaitForEvents(1*time.Second, rollbar.Close, authentication.LegacyClose); err != nil {
			logging.Warning("Failed to wait for rollbar to close: %v", err)
		}
		os.Exit(exitCode)
	}()

	// Set up verbose logging
	logging.CurrentHandler().SetVerbose(os.Getenv("VERBOSE") != "")
	// Set up rollbar reporting
	logging.SetupRollbar(constants.StateInstallerRollbarToken)

	// Set up configuration handler
	cfg, err := config.New()
	if err != nil {
		logging.Error("Could not set up configuration handler: " + errs.JoinMessage(err))
		fmt.Fprintln(os.Stderr, err.Error())
		exitCode = 1
	}
	defer cfg.Close()

	// Set up machineid, allowing us to anonymously group errors and analytics
	machineid.Configure(cfg)
	machineid.SetErrorLogger(logging.Error)

	// Set up analytics. We make an effort to anonymize all analytics, but analytics are crucial to enhance the product.
	analytics.Configure(cfg)

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

	var garbageBool bool

	params := newParams()
	cmd := captain.NewCommand(
		"state-installer",
		"",
		"Installs or updates the State Tool",
		out,
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
				Name:        "command",
				Shorthand:   "c",
				Description: "Run any command after the install script has completed",
				Value:       &params.command,
			},
			{
				Name:        "activate",
				Description: "Activate a project when State Tool is correctly installed",
				Value:       params.activate,
			},
			{
				Name:        "activate-default",
				Description: "Activate a project and make it the system default",
				Value:       params.activateDefault,
			},
			{
				Name:        "version",
				Shorthand:   "v",
				Description: "The version of the State Tool to install",
				Value:       &params.version,
			},
			{
				Name:   "source-path",
				Hidden: true, // Source path should ideally only be used through state tool updates (ie. it's internally routed)
				Value:  &params.sourcePath,
			},
			{
				Name:   "from-deferred",
				Hidden: true, // This is set when deferring installs to another installer, to avoid redundant UI
				Value:  &params.fromDeferred,
			},
			{
				Name:      "path",
				Shorthand: "t",
				Hidden:    true, // Since we already expose the path as an argument, let's not confuse the user
				Value:     &params.path,
			},
			// The remaining flags are for backwards compatibility (ie. we don't want to error out when they're provided)
			{Name: "nnn", Shorthand: "n", Hidden: true, Value: &garbageBool}, // don't prompt; useless cause we don't prompt anyway
			{Name: "fff", Shorthand: "f", Hidden: true, Value: &garbageBool}, // overwrite existing state tool; useless as that's already the default
		},
		[]*captain.Argument{
			{
				Name:        "path",
				Description: "Install into target directory <path>",
				Value:       &params.path,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			return execute(out, cfg, args, params)
		},
	)

	args := os.Args

	// We have old install one liners around that use `-activate` instead of `--activate`
	for _, v := range args {
		if strings.HasPrefix(v, "-activate") {
			v = "--activate" + strings.TrimPrefix(v, "-activate")
		}
	}

	if err := cmd.Execute(args[1:]); err != nil {
		logging.Error(errs.JoinMessage(err))
		out.Error(err.Error())
		exitCode = 1
		return
	}
}

func execute(out output.Outputer, cfg *config.Instance, args []string, params *Params) error {
	// if sourcePath was provided we're already using the right installer, so proceed with installation
	if params.sourcePath != "" {
		return installFromLocalSource(out, cfg, args, params)
	}

	// If no sourcePath was provided then we still need to download the source files, and defer the actual
	// installation to the installer contained within the source file
	return installFromRemoteSource(out, cfg, args, params)
}

// installFromLocalSource is invoked when we're performing an installation where the payload is already provided
func installFromLocalSource(out output.Outputer, cfg *config.Instance, args []string, params *Params) error {
	installer, err := NewInstaller(cfg, out, params)
	if err != nil {
		out.Print("[ERROR]x Failed[/RESET]")
		return err
	}
	out.Fprint(os.Stdout, fmt.Sprintf("• Installing State Tool to [NOTICE]%s[/RESET]... ", installer.InstallPath()))

	// Run installer
	if err := installer.Install(); err != nil {
		out.Print("[ERROR]x Failed[/RESET]")
		return err
	}
	out.Print("[SUCCESS]✔ Done[/RESET]")

	// Execute requested command, these are mutually exclusive
	switch {
	// Execute provided --command
	case params.command != "":
		out.Print(output.Heading(fmt.Sprintf("Running `[NOTICE]%s[/RESET]`", params.command)))
		cmd, args := exeutils.DecodeCmd(params.command)
		if _, _, err := exeutils.ExecuteAndPipeStd(cmd, args, []string{}); err != nil {
			return errs.Wrap(err, "Running provided command failed")
		}
	// Activate provided --activate Namespace
	case params.activate.IsValid():
		if _, _, err := exeutils.ExecuteAndPipeStd("state", []string{"activate", params.activate.String()}, []string{}); err != nil {
			return errs.Wrap(err, "Could not activate %s", params.activate.String())
		}
	// Activate provided --activate-default Namespace
	case params.activateDefault.IsValid():
		if _, _, err := exeutils.ExecuteAndPipeStd("state", []string{"activate", params.activateDefault.String(), "--default"}, []string{}); err != nil {
			return errs.Wrap(err, "Could not activate %s", params.activateDefault.String())
		}
	default:
		out.Print("")
		out.Print(output.Title("Installation Complete"))
		out.Print("")
		out.Print("State Tool Package Manager has been successfully installed. You may need to start a new shell to start using it.")
	}

	return nil
}

// installFromRemoteSource is invoked when we run the installer without providing the associated source files
// Effectively this will download and unpack the target version and then run the installer packaged for that version
// To view the source of the target version you can extract the relevant commit ID from the version of the target version
// This is the default behavior when doing a clean install
func installFromRemoteSource(out output.Outputer, cfg *config.Instance, args []string, params *Params) error {
	out.Print(output.Title("Installing State Tool Package Manager\n"))
	out.Print(`The State Tool lets you install and manage your language runtimes.` + "\n" +
		`ActiveState collects usage statistics and diagnostic data about failures. ` + "\n" +
		`By using the State Tool Package Manager you agree to the terms of ActiveState’s Privacy Policy, ` +
		`available at: [ACTIONABLE]https://www.activestate.com/company/privacy-policy[/RESET]`)

	args = append(args, "--from-deferred")

	// Fetch payload
	checker := updater.NewDefaultChecker(cfg)
	checker.VerifyVersion = false
	update, err := checker.CheckFor(params.branch, params.version)
	if err != nil {
		return errs.Wrap(err, "Could not retrieve install package information")
	}
	if update == nil {
		return errs.Wrap(err, "No update information could be found.")
	}

	version := update.Version
	if params.branch != "" {
		version = fmt.Sprintf("%s (%s)", version, params.branch)
	}

	out.Fprint(os.Stdout, fmt.Sprintf("• Downloading State Tool version [NOTICE]%s[/RESET]... ", version))
	if _, err := update.DownloadAndUnpack(); err != nil {
		out.Print("[ERROR]x Failed[/RESET]")
		return errs.Wrap(err, "Could not download and unpack")
	}
	out.Print("[SUCCESS]✔ Done[/RESET]")

	return update.InstallBlocking("", args...)
}
