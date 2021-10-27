package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics/service"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/rollbar/rollbar-go"
)

const AnalyticsCat = "installer"
const AnalyticsFunnelCat = "installer-funnel"

type Params struct {
	fromDeferred    bool
	sourcePath      string
	sourceInstaller string
	path            string
	updateTag       string
	branch          string
	command         string
	version         string
	force           bool
	activate        *project.Namespaced
	activateDefault *project.Namespaced
}

func newParams() *Params {
	return &Params{activate: &project.Namespaced{}, activateDefault: &project.Namespaced{}}
}

func main() {
	var exitCode int

	var an *service.Analytics

	// Handle things like panics, exit codes and the closing of globals
	defer func() {
		if panics.HandlePanics(recover(), debug.Stack()) {
			exitCode = 1
		}
		if err := events.WaitForEvents(5*time.Second, rollbar.Close, an.Wait); err != nil {
			logging.Error("state-installer failed to wait for events: %v", err)
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

	// We have old install one liners around that use `-activate` instead of `--activate`
	processedArgs := os.Args
	for x, v := range processedArgs {
		if strings.HasPrefix(v, "-activate") {
			processedArgs[x] = "--activate" + strings.TrimPrefix(v, "-activate")
		}
	}

	logging.Debug("Original Args: %v", os.Args)
	logging.Debug("Processed Args: %v", processedArgs)

	an = service.NewAnalytics()
	an.Configure(cfg, nil)

	an.Event(AnalyticsFunnelCat, "start")

	logging.SetupRollbarReporter(func(msg string) { an.Event("rollbar", msg) })

	params := newParams()
	cmd := captain.NewCommand(
		"state-installer",
		"",
		"Installs or updates the State Tool",
		primer.New(nil, out, nil, nil, nil, nil, cfg, nil, an),
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
				Name:        "force",
				Shorthand:   "f",
				Description: "Force the installation, overwriting any version of the State Tool already installed",
				Value:       &params.force,
			},
			{
				Name:   "source-installer",
				Hidden: true, // This is internally routed in via the install frontend (eg. install.sh, MSI, etc)
				Value:  &params.sourceInstaller,
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
		},
		[]*captain.Argument{
			{
				Name:        "path",
				Description: "Install into target directory <path>",
				Value:       &params.path,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return execute(out, cfg, an, processedArgs[1:], params)
		},
	)

	an.Event(AnalyticsFunnelCat, "pre-exec")
	err = cmd.Execute(processedArgs[1:])
	if err != nil {
		if locale.IsInputError(err) {
			an.EventWithLabel(AnalyticsCat, "input-error", errs.JoinMessage(err))
			logging.Error("Installer input error: " + errs.JoinMessage(err))
		} else {
			an.EventWithLabel(AnalyticsCat, "error", errs.JoinMessage(err))
			logging.Critical("Installer error: " + errs.JoinMessage(err))
		}
		an.EventWithLabel(AnalyticsFunnelCat, "fail", err.Error())
		out.Error(err.Error())
		exitCode = 1
		return
	}
	an.Event(AnalyticsFunnelCat, "success")
}

func execute(out output.Outputer, cfg *config.Instance, an *service.Analytics, args []string, params *Params) error {
	an.Event(AnalyticsFunnelCat, "exec")

	// Detect installed state tool
	stateToolInstalled, err := installation.InstalledOnPath(params.path)
	if err != nil {
		return errs.Wrap(err, "Could not detect if State Tool is already installed.")
	}

	// Detect state tool alongside installer executable
	installerPath := filepath.Dir(osutils.Executable())
	packagedStateExe := appinfo.StateApp(installerPath).Exec()

	// Detect whether this is a fresh install or an update
	isUpdate := false
	switch {
	case params.force:
		logging.Debug("Not using update flow as --force was passed")
		break // When ran with `--force` we always use the install UX
	case fileutils.FileExists(packagedStateExe):
		logging.Debug("Using update flow as installer is alongside payload")
		isUpdate = true

		if params.sourcePath == "" {
			// Older versions of state tool do not invoke the installer with `--source-path`
			params.sourcePath = installerPath
		}
	case stateToolInstalled:
		// This should trigger AFTER the check above where sourcePath is defined
		logging.Debug("Using update flow as state tool is already installed")
		isUpdate = true
	}

	route := "install"
	if isUpdate {
		route = "update"
	}
	an.Event(AnalyticsFunnelCat, route)

	// if sourcePath was provided we're already using the right installer, so proceed with installation
	if params.sourcePath != "" {
		if err := installOrUpdateFromLocalSource(out, cfg, an, params, isUpdate); err != nil {
			return err
		}
		return postInstallEvents(out, an, params)
	}

	// Check if state tool already installed
	if !params.force && stateToolInstalled {
		logging.Debug("Cancelling out because State Tool is already installed")
		out.Print("State Tool Package Manager is already installed. To reinstall use the [ACTIONABLE]--force[/RESET] flag.")
		an.Event(AnalyticsFunnelCat, "already-installed")
		return postInstallEvents(out, an, params)
	}

	// If no sourcePath was provided then we still need to download the source files, and defer the actual
	// installation to the installer contained within the source file
	return installFromRemoteSource(out, cfg, an, args, params)
}

// installOrUpdateFromLocalSource is invoked when we're performing an installation where the payload is already provided
func installOrUpdateFromLocalSource(out output.Outputer, cfg *config.Instance, an *service.Analytics, params *Params, isUpdate bool) error {
	logging.Debug("Install from local source")
	an.Event(AnalyticsFunnelCat, "local-source")

	installer, err := NewInstaller(cfg, out, params)
	if err != nil {
		out.Print(fmt.Sprintf("[ERROR]Could not create installer: %s[/RESET]", errs.JoinMessage(err)))
		return err
	}

	if isUpdate {
		out.Fprint(os.Stdout, "• Installing Update... ")
	} else {
		out.Fprint(os.Stdout, fmt.Sprintf("• Installing State Tool to [NOTICE]%s[/RESET]... ", installer.InstallPath()))
	}

	// Run installer
	an.Event(AnalyticsFunnelCat, "pre-installer")
	if err := installer.Install(); err != nil {
		out.Print("[ERROR]x Failed[/RESET]")
		return err
	}
	an.Event(AnalyticsFunnelCat, "post-installer")
	out.Print("[SUCCESS]✔ Done[/RESET]")

	if !isUpdate {
		out.Print("")
		out.Print(output.Title("State Tool Package Manager Installation Complete"))
		out.Print("State Tool Package Manager has been successfully installed. You may need to start a new shell to start using it.")
	}

	return nil
}

func postInstallEvents(out output.Outputer, an *service.Analytics, params *Params) error {
	an.Event(AnalyticsFunnelCat, "post-install-events")

	installPath, err := resolveInstallPath(params.path)
	if err != nil {
		return errs.Wrap(err, "Could not resolve installation path")
	}

	stateExe := appinfo.StateApp(installPath).Exec()
	binPath, err := installation.BinPathFromInstallPath(installPath)
	if err != nil {
		return errs.Wrap(err, "Could not detect installation bin path")
	}
	env := []string{"PATH=" + binPath + string(os.PathListSeparator) + os.Getenv("PATH")}

	// Execute requested command, these are mutually exclusive
	switch {
	// Execute provided --command
	case params.command != "":
		an.Event(AnalyticsFunnelCat, "forward-command")

		out.Print(fmt.Sprintf("\nRunning `[ACTIONABLE]%s[/RESET]`\n", params.command))
		cmd, args := exeutils.DecodeCmd(params.command)
		if _, _, err := exeutils.ExecuteAndPipeStd(cmd, args, env); err != nil {
			an.EventWithLabel(AnalyticsFunnelCat, "forward-command-err", err.Error())
			return errs.Wrap(err, "Running provided command failed, error returned: %s", errs.JoinMessage(err))
		}
	// Activate provided --activate Namespace
	case params.activate.IsValid():
		an.Event(AnalyticsFunnelCat, "forward-activate")

		out.Print(fmt.Sprintf("\nRunning `[ACTIONABLE]state activate %s[/RESET]`\n", params.activate.String()))
		if _, _, err := exeutils.ExecuteAndPipeStd(stateExe, []string{"activate", params.activate.String()}, env); err != nil {
			an.EventWithLabel(AnalyticsFunnelCat, "forward-activate-err", err.Error())
			return errs.Wrap(err, "Could not activate %s, error returned: %s", params.activate.String(), errs.JoinMessage(err))
		}
	// Activate provided --activate-default Namespace
	case params.activateDefault.IsValid():
		an.Event(AnalyticsFunnelCat, "forward-activate-default")

		out.Print(fmt.Sprintf("\nRunning `[ACTIONABLE]state activate --default %s[/RESET]`\n", params.activateDefault.String()))
		if _, _, err := exeutils.ExecuteAndPipeStd(stateExe, []string{"activate", params.activateDefault.String(), "--default"}, env); err != nil {
			an.EventWithLabel(AnalyticsFunnelCat, "forward-activate-default-err", err.Error())
			return errs.Wrap(err, "Could not activate %s, error returned: %s", params.activateDefault.String(), errs.JoinMessage(err))
		}
	}

	return nil
}

// installFromRemoteSource is invoked when we run the installer without providing the associated source files
// Effectively this will download and unpack the target version and then run the installer packaged for that version
// To view the source of the target version you can extract the relevant commit ID from the version of the target version
// This is the default behavior when doing a clean install
func installFromRemoteSource(out output.Outputer, cfg *config.Instance, an *service.Analytics,args []string, params *Params) error {
	an.Event(AnalyticsFunnelCat, "local-source")

	out.Print(output.Title("Installing State Tool Package Manager"))
	out.Print(`The State Tool lets you install and manage your language runtimes.` + "\n\n" +
		`ActiveState collects usage statistics and diagnostic data about failures. ` + "\n" +
		`By using the State Tool Package Manager you agree to the terms of ActiveState’s Privacy Policy, ` + "\n" +
		`available at: [ACTIONABLE]https://www.activestate.com/company/privacy-policy[/RESET]` + "\n")

	args = append(args, "--from-deferred")

	storeInstallSource(params.sourceInstaller)

	// Fetch payload
	checker := updater.NewDefaultChecker(cfg)
	checker.InvocationSource = updater.InvocationSourceInstall // Installing from a remote source is only ever encountered via the install flow
	checker.VerifyVersion = false
	update, err := checker.CheckFor(params.branch, params.version)
	if err != nil {
		return errs.Wrap(err, "Could not retrieve install package information")
	}
	if update == nil {
		return errs.New("No update information could be found.")
	}

	version := update.Version
	if params.branch != "" {
		version = fmt.Sprintf("%s (%s)", version, params.branch)
	}

	an.Event(AnalyticsFunnelCat, "download")
	out.Fprint(os.Stdout, fmt.Sprintf("• Downloading State Tool version [NOTICE]%s[/RESET]... ", version))
	if _, err := update.DownloadAndUnpack(); err != nil {
		out.Print("[ERROR]x Failed[/RESET]")
		return errs.Wrap(err, "Could not download and unpack")
	}
	out.Print("[SUCCESS]✔ Done[/RESET]")

	cfg.Set(updater.CfgKeyInstallVersion, params.version)

	an.Event(AnalyticsFunnelCat, "install-async")
	return update.InstallBlocking(params.path, args...)
}

// storeInstallSource writes the name of the install client (eg. install.sh) to the appdata dir
// this is used in analytics to give us a sense for where our users are coming from
func storeInstallSource(installSource string) {
	if installSource == "" {
		installSource = "state-installer"
	}

	appData, err := storage.AppDataPath()
	if err != nil {
		logging.Error("Could not store install source due to AppDataPath error: %s", errs.JoinMessage(err))
		return
	}
	if err := fileutils.WriteFile(filepath.Join(appData, constants.InstallSourceFile), []byte(installSource)); err != nil {
		logging.Error("Could not store install source due to WriteFile error: %s", errs.JoinMessage(err))
	}
}

func resolveInstallPath(path string) (string, error) {
	if path != "" {
		return filepath.Abs(path)
	} else {
		return installation.InstallPath()
	}
}
