package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/client/sync"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/runbits/errors"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"golang.org/x/term"
)

type Params struct {
	sourceInstaller string
	path            string
	updateTag       string
	command         string
	force           bool
	isUpdate        bool
	activate        *project.Namespaced
	activateDefault *project.Namespaced
	showVersion     bool
	nonInteractive  bool
}

func newParams() *Params {
	return &Params{
		activate:        &project.Namespaced{},
		activateDefault: &project.Namespaced{},
		nonInteractive:  !term.IsTerminal(int(os.Stdin.Fd())),
	}
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

		if cfg != nil {
			events.Close("config", cfg.Close)
		}

		if err := events.WaitForEvents(5*time.Second, rollbar.Wait, an.Wait, logging.Close); err != nil {
			logging.Warning("state-installer failed to wait for events: %v", err)
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
	var err error
	cfg, err = config.New()
	if err != nil {
		multilog.Error("Could not set up configuration handler: " + errs.JoinMessage(err))
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
		multilog.Error("Could not set up output handler: " + errs.JoinMessage(err))
		fmt.Fprintln(os.Stderr, err.Error())
		exitCode = 1
		return
	}

	var garbageString string

	// We have old install one liners around that use `-activate` instead of `--activate`
	processedArgs := os.Args
	for x, v := range processedArgs {
		if strings.HasPrefix(v, "-activate") {
			processedArgs[x] = "--activate" + strings.TrimPrefix(v, "-activate")
		}
	}

	logging.Debug("Original Args: %v", os.Args)
	logging.Debug("Processed Args: %v", processedArgs)

	// Store sessionToken to config
	for _, envVar := range []string{constants.OverrideSessionTokenEnvVarName, constants.SessionTokenEnvVarName} {
		sessionToken, ok := os.LookupEnv(envVar)
		if !ok {
			continue
		}
		err := cfg.Set(anaConst.CfgSessionToken, sessionToken)
		if err != nil {
			multilog.Error("Unable to set session token: " + errs.JoinMessage(err))
		}
		break
	}

	an = sync.New(anaConst.SrcStateInstaller, cfg, nil, out)
	an.Event(anaConst.CatInstallerFunnel, "start")

	params := newParams()
	cmd := captain.NewCommand(
		"state-installer",
		"",
		"Installs or updates the State Tool",
		primer.New(nil, out, nil, nil, nil, nil, cfg, nil, nil, an),
		[]*captain.Flag{ // The naming of these flags is slightly inconsistent due to backwards compatibility requirements
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
				Description: "Activate a project and make it always available for use",
				Value:       params.activateDefault,
			},
			{
				Name:        "force",
				Shorthand:   "f",
				Description: "Force the installation, overwriting any version of the State Tool already installed",
				Value:       &params.force,
			},
			{
				Name:        "update",
				Shorthand:   "u",
				Description: "Force update behaviour for the installer",
				Value:       &params.isUpdate,
			},
			{
				Name:   "source-installer",
				Hidden: true, // This is internally routed in via the install frontend (eg. install.sh, etc)
				Value:  &params.sourceInstaller,
			},
			{
				Name:      "path",
				Shorthand: "t",
				Hidden:    true, // Since we already expose the path as an argument, let's not confuse the user
				Value:     &params.path,
			},
			{
				Name:  "version", // note: no shorthand because install.sh uses -v for selecting version
				Value: &params.showVersion,
			},
			{Name: "non-interactive", Shorthand: "n", Hidden: true, Value: &params.nonInteractive}, // don't prompt
			// The remaining flags are for backwards compatibility (ie. we don't want to error out when they're provided)
			{Name: "channel", Hidden: true, Value: &garbageString},
			{Name: "bbb", Shorthand: "b", Hidden: true, Value: &garbageString},
			{Name: "vvv", Shorthand: "v", Hidden: true, Value: &garbageString},
			{Name: "source-path", Hidden: true, Value: &garbageString},
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

	an.Event(anaConst.CatInstallerFunnel, "pre-exec")
	err = cmd.Execute(processedArgs[1:])
	if err != nil {
		errors.ReportError(err, cmd, an)
		if locale.IsInputError(err) {
			an.EventWithLabel(anaConst.CatInstaller, "input-error", errs.JoinMessage(err))
			logging.Debug("Installer input error: " + errs.JoinMessage(err))
		} else {
			an.EventWithLabel(anaConst.CatInstaller, "error", errs.JoinMessage(err))
			multilog.Critical("Installer error: " + errs.JoinMessage(err))
		}

		an.EventWithLabel(anaConst.CatInstallerFunnel, "fail", errs.JoinMessage(err))
		exitCode, err = errors.ParseUserFacing(err)
		if err != nil {
			out.Error(err)
		}
	} else {
		an.Event(anaConst.CatInstallerFunnel, "success")
	}
}

func execute(out output.Outputer, cfg *config.Instance, an analytics.Dispatcher, args []string, params *Params) error {
	if params.showVersion {
		vd := installation.VersionData{
			"CLI Installer",
			constants.LibraryLicense,
			constants.Version,
			constants.ChannelName,
			constants.RevisionHash,
			constants.Date,
			constants.OnCI == "true",
		}
		out.Print(locale.T("version_info", vd))
		return nil
	}

	an.Event(anaConst.CatInstallerFunnel, "exec")

	if params.path == "" {
		var err error
		params.path, err = installation.InstallPathForChannel(constants.ChannelName)
		if err != nil {
			return errs.Wrap(err, "Could not detect installation path.")
		}
	}

	// Detect installed state tool
	stateToolInstalled, installPath, err := installedOnPath(params.path, constants.ChannelName)
	if err != nil {
		return errs.Wrap(err, "Could not detect if State Tool is already installed.")
	}
	if stateToolInstalled && installPath != params.path {
		logging.Debug("Setting path to: %s", installPath)
		params.path = installPath
	}

	// Detect if target dir is existing install of same target channel
	var installedChannel string
	marker := filepath.Join(installPath, installation.InstallDirMarker)
	if stateToolInstalled && fileutils.TargetExists(marker) {
		markerContents, err := fileutils.ReadFile(marker)
		if err != nil {
			return errs.Wrap(err, "Could not read marker file")
		}
		// The marker file is empty for versions prior to v0.40.0-RC3
		if len(markerContents) > 0 {
			var markerMeta installation.InstallMarkerMeta
			if err := json.Unmarshal(markerContents, &markerMeta); err != nil {
				return errs.Wrap(err, "Could not parse install marker file")
			}
			installedChannel = markerMeta.Channel
		}
	}
	// Older state tools did not bake in meta information, in this case we allow overwriting regardless of channel
	targetingSameChannel := installedChannel == "" || installedChannel == constants.ChannelName
	stateToolInstalledAndFunctional := stateToolInstalled && installationIsOnPATH(params.path) && targetingSameChannel

	// If this is a fresh installation we ensure that the target directory is empty
	if !stateToolInstalled && fileutils.DirExists(params.path) && !params.force {
		contains, err := fileutils.DirContains(params.path, installation.InstallDirMarker)
		if err != nil {
			return errs.Wrap(err, "Could not check if install path is empty")
		}
		if !contains {
			return locale.NewError("err_install_nonempty_dir", "Installation path must be an empty directory: {{.V0}}", params.path)
		}
	}

	// We expect the installer payload to be in the same directory as the installer itself
	payloadPath := filepath.Dir(osutils.Executable())

	route := "install"
	if params.isUpdate {
		route = "update"
	}
	an.Event(anaConst.CatInstallerFunnel, route)

	// Check if state tool already installed and functional
	if stateToolInstalledAndFunctional && !params.isUpdate && !params.force {
		logging.Debug("Cancelling out because State Tool is already installed and functional")
		out.Print(fmt.Sprintf("State Tool Package Manager is already installed at [NOTICE]%s[/RESET]. To reinstall use the [ACTIONABLE]--force[/RESET] flag.", installPath))
		an.Event(anaConst.CatInstallerFunnel, "already-installed")
		params.isUpdate = true
		return postInstallEvents(out, cfg, an, params)
	}

	if err := installOrUpdateFromLocalSource(out, cfg, an, payloadPath, params); err != nil {
		return err
	}
	storeInstallSource(params.sourceInstaller)
	return postInstallEvents(out, cfg, an, params)
}

// installOrUpdateFromLocalSource is invoked when we're performing an installation where the payload is already provided
func installOrUpdateFromLocalSource(out output.Outputer, cfg *config.Instance, an analytics.Dispatcher, payloadPath string, params *Params) error {
	logging.Debug("Install from local source")
	an.Event(anaConst.CatInstallerFunnel, "local-source")
	if !params.isUpdate {
		// install.sh or install.ps1 downloaded this installer and is running it.
		out.Print(output.Title("Installing State Tool Package Manager"))
		out.Print(`The State Tool lets you install and manage your language runtimes.` + "\n\n" +
			`ActiveState collects usage statistics and diagnostic data about failures. ` + "\n" +
			`By using the State Tool Package Manager you agree to the terms of ActiveState’s Privacy Policy, ` + "\n" +
			`available at: [ACTIONABLE]https://www.activestate.com/company/privacy-policy[/RESET]` + "\n")
	}

	if err := assertCompatibility(); err != nil {
		// Don't wrap, we want the error from assertCompatibility to be returned -- installer doesn't have intelligent error handling yet
		// https://activestatef.atlassian.net/browse/DX-957
		return err
	}

	installer, err := NewInstaller(cfg, out, an, payloadPath, params)
	if err != nil {
		out.Print(fmt.Sprintf("[ERROR]Could not create installer: %s[/RESET]", errs.JoinMessage(err)))
		return err
	}

	if params.isUpdate {
		out.Fprint(os.Stdout, "• Installing Update... ")
	} else {
		out.Fprint(os.Stdout, fmt.Sprintf("• Installing State Tool to [NOTICE]%s[/RESET]... ", installer.InstallPath()))
	}

	// Run installer
	an.Event(anaConst.CatInstallerFunnel, "pre-installer")
	if err := installer.Install(); err != nil {
		out.Print("[ERROR]x Failed[/RESET]")
		return err
	}
	an.Event(anaConst.CatInstallerFunnel, "post-installer")
	out.Print("[SUCCESS]✔ Done[/RESET]")

	if !params.isUpdate {
		out.Print("")
		out.Print(output.Title("State Tool Package Manager Installation Complete"))
		out.Print("State Tool Package Manager has been successfully installed.")
	}

	return nil
}

func postInstallEvents(out output.Outputer, cfg *config.Instance, an analytics.Dispatcher, params *Params) error {
	an.Event(anaConst.CatInstallerFunnel, "post-install-events")

	installPath, err := resolveInstallPath(params.path)
	if err != nil {
		return errs.Wrap(err, "Could not resolve installation path")
	}

	binPath, err := installation.BinPathFromInstallPath(installPath)
	if err != nil {
		return errs.Wrap(err, "Could not detect installation bin path")
	}

	stateExe, err := installation.StateExecFromDir(installPath)
	if err != nil {
		return locale.WrapError(err, "err_state_exec")
	}

	ss := subshell.New(cfg)
	if ss.Shell() == bash.Name && runtime.GOOS == "darwin" {
		out.Print(locale.T("warning_macos_bash"))
	}

	// Execute requested command, these are mutually exclusive
	switch {
	// Execute provided --command
	case params.command != "":
		an.Event(anaConst.CatInstallerFunnel, "forward-command")

		out.Print(fmt.Sprintf("\nRunning '[ACTIONABLE]%s[/RESET]'\n", params.command))
		cmd, args := osutils.DecodeCmd(params.command)
		if _, _, err := osutils.ExecuteAndPipeStd(cmd, args, envSlice(binPath)); err != nil {
			an.EventWithLabel(anaConst.CatInstallerFunnel, "forward-command-err", err.Error())
			return errs.Silence(errs.Wrap(err, "Running provided command failed, error returned: %s", errs.JoinMessage(err)))
		}
	// Activate provided --activate Namespace
	case params.activate.IsValid():
		an.Event(anaConst.CatInstallerFunnel, "forward-activate")

		out.Print(fmt.Sprintf("\nRunning '[ACTIONABLE]state activate %s[/RESET]'\n", params.activate.String()))
		if _, _, err := osutils.ExecuteAndPipeStd(stateExe, []string{"activate", params.activate.String()}, envSlice(binPath)); err != nil {
			an.EventWithLabel(anaConst.CatInstallerFunnel, "forward-activate-err", err.Error())
			return errs.Silence(errs.Wrap(err, "Could not activate %s, error returned: %s", params.activate.String(), errs.JoinMessage(err)))
		}
	// Activate provided --activate-default Namespace
	case params.activateDefault.IsValid():
		an.Event(anaConst.CatInstallerFunnel, "forward-activate-default")

		out.Print(fmt.Sprintf("\nRunning '[ACTIONABLE]state activate --default %s[/RESET]'\n", params.activateDefault.String()))
		if _, _, err := osutils.ExecuteAndPipeStd(stateExe, []string{"activate", params.activateDefault.String(), "--default"}, envSlice(binPath)); err != nil {
			an.EventWithLabel(anaConst.CatInstallerFunnel, "forward-activate-default-err", err.Error())
			return errs.Silence(errs.Wrap(err, "Could not activate %s, error returned: %s", params.activateDefault.String(), errs.JoinMessage(err)))
		}
	case !params.isUpdate && term.IsTerminal(int(os.Stdin.Fd())) && os.Getenv(constants.InstallerNoSubshell) != "true" && os.Getenv("TERM") != "dumb":
		if err := ss.SetEnv(osutils.InheritEnv(envMap(binPath))); err != nil {
			return locale.WrapError(err, "err_subshell_setenv")
		}
		if err := ss.Activate(nil, cfg, out); err != nil {
			return errs.Wrap(err, "Error activating subshell: %s", errs.JoinMessage(err))
		}
		if err = <-ss.Errors(); err != nil && !errs.IsSilent(err) {
			return errs.Wrap(err, "Error during subshell execution: %s", errs.JoinMessage(err))
		}
	}

	return nil
}

func envSlice(binPath string) []string {
	return []string{
		"PATH=" + binPath + string(os.PathListSeparator) + os.Getenv("PATH"),
		constants.DisableErrorTipsEnvVarName + "=true",
	}
}

func envMap(binPath string) map[string]string {
	return map[string]string{
		"PATH": binPath + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
}

// storeInstallSource writes the name of the install client (eg. install.sh) to the appdata dir
// this is used in analytics to give us a sense for where our users are coming from
func storeInstallSource(installSource string) {
	if installSource == "" {
		installSource = "state-installer"
	}

	appData, err := storage.AppDataPath()
	if err != nil {
		multilog.Error("Could not store install source due to AppDataPath error: %s", errs.JoinMessage(err))
		return
	}
	if err := fileutils.WriteFile(filepath.Join(appData, constants.InstallSourceFile), []byte(installSource)); err != nil {
		multilog.Error("Could not store install source due to WriteFile error: %s", errs.JoinMessage(err))
	}
}

func resolveInstallPath(path string) (string, error) {
	if path != "" {
		return filepath.Abs(path)
	} else {
		return installation.DefaultInstallPath()
	}
}

func assertCompatibility() error {
	if sysinfo.OS() == sysinfo.Windows {
		osv, err := sysinfo.OSVersion()
		if err != nil {
			return locale.WrapError(err, "windows_compatibility_warning", "", err.Error())
		} else if osv.Major < 10 || (osv.Major == 10 && osv.Micro < 17134) {
			return locale.WrapError(err, "windows_compatibility_error", "", osv.Name, osv.Version)
		}
	}

	return nil
}
