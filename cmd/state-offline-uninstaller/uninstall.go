package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	ac "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/project"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

const licenseFileName = "LICENSE.txt"

type runner struct {
	out       output.Outputer
	prompt    prompt.Prompter
	analytics analytics.Dispatcher
	cfg       *config.Instance
	shell     subshell.SubShell
	icfg      InstallerConfig
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Analyticer
	primer.Configurer
	primer.Subsheller
}

func NewRunner(prime primeable) *runner {
	return &runner{
		prime.Output(),
		prime.Prompt(),
		prime.Analytics(),
		prime.Config(),
		prime.Subshell(),
		InstallerConfig{},
	}
}

const installerConfigFileName = "installer_config.json"

type InstallerConfig struct {
	OrgName     string `json:"org_name"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	CommitID    string `json:"commit_id"`
}

type Params struct {
	path string
}

func newParams() *Params {
	return &Params{}
}

func (r *runner) Run(params *Params) (rerr error) {
	var installerDimensions *dimensions.Values
	defer func() {
		if rerr == nil {
			return
		}
		if locale.IsInputError(rerr) {
			r.analytics.EventWithLabel(ac.CatOfflineInstaller, ac.ActOfflineInstallerAbort, errs.JoinMessage(rerr), installerDimensions)
		} else {
			r.analytics.EventWithLabel(ac.CatOfflineInstaller, ac.ActOfflineInstallerFailure, errs.JoinMessage(rerr), installerDimensions)
		}
	}()

	// Detect target path
	targetPath, err := r.getTargetPath(params.path)
	if err != nil {
		return errs.Wrap(err, "Could not determine target path")
	}

	/* Validate Target Path */
	if err := r.validateTargetPath(targetPath); err != nil {
		return errs.Wrap(err, "Could not validate target path")
	}

	if err := r.prepareInstallerConfig(targetPath); err != nil {
		return errs.Wrap(err, "Could not read installer config, this installer appears to be corrupted.")
	}

	cont, err := r.prompt.Confirm("",
		fmt.Sprintf("You are about to uninstall the runtime installed at %s, continue?", targetPath),
		p.BoolP(false))
	if err != nil {
		return errs.Wrap(err, "Could not confirm uninstall")
	}
	if !cont {
		return locale.NewInputError("err_uninstall_abort", "Uninstall aborted")
	}

	installerDimensions = &dimensions.Values{
		ProjectNameSpace: p.StrP(project.NewNamespace(r.icfg.OrgName, r.icfg.ProjectName, "").String()),
		CommitID:         &r.icfg.CommitID,
		Trigger:          p.StrP(target.TriggerOfflineUninstaller.String()),
	}
	r.analytics.Event(ac.CatOfflineInstaller, ac.ActOfflineInstallerStart, installerDimensions)

	r.out.Print("Removing environment configuration")
	err = r.removeEnvPaths()
	if err != nil {
		return errs.Wrap(err, "Error removing environment path")
	}

	r.out.Print("Removing installation directory")
	err = os.RemoveAll(targetPath)
	if err != nil {
		return errs.Wrap(err, "Error removing installation directory")
	}

	r.analytics.Event(ac.CatOfflineInstaller, ac.ActOfflineInstallerSuccess, installerDimensions)
	r.analytics.Event(ac.CatRuntimeUsage, ac.ActRuntimeDelete, installerDimensions)

	r.out.Print("Uninstall Complete")

	return nil
}

func (r *runner) prepareInstallerConfig(assetsPath string) error {
	icfg := InstallerConfig{}
	installerConfigPath := filepath.Join(assetsPath, installerConfigFileName)

	configData, err := os.ReadFile(installerConfigPath)
	if err != nil {
		return errs.Wrap(err, "Failed to read config_file")
	}
	if err := json.Unmarshal(configData, &icfg); err != nil {
		return errs.Wrap(err, "Failed to decode config_file")
	}

	if icfg.ProjectName == "" {
		return errs.New("ProjectName is empty")
	}

	if icfg.OrgName == "" {
		return errs.New("OrgName is empty")
	}

	if icfg.CommitID == "" {
		return errs.New("CommitID is empty")
	}

	r.icfg = icfg

	return nil
}

func (r *runner) getTargetPath(inputPath string) (string, error) {
	if inputPath != "" {
		return inputPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", errs.Wrap(err, "Could not determine current working directory")
	}

	var targetPath string
	if fileutils.TargetExists(filepath.Join(cwd, installerConfigFileName)) {
		targetPath = cwd
	}

	if targetPath != "" {
		targetPath, err = r.prompt.Input("", "Enter an installation directory to uninstall", &targetPath)
	} else {
		targetPath, err = r.prompt.Input("", "Enter an installation directory to uninstall", nil, prompt.InputRequired)
	}
	if err != nil {
		return "", errs.Wrap(err, "Could not retrieve installation directory")
	}
	return targetPath, nil
}

func (r *runner) validateTargetPath(path string) error {
	if !fileutils.IsWritable(path) {
		return errs.New(
			"Cannot write to [ACTIONABLE]%s[/RESET]. Please ensure that the directory is writeable without "+
				"needing admin privileges or run this installer with Admin.", path)
	}

	if !fileutils.IsDir(path) {
		return errs.New("Target path [ACTIONABLE]%s[/RESET] is not a directory", path)
	}

	installerConfigPath := filepath.Join(path, installerConfigFileName)
	if !fileutils.FileExists(installerConfigPath) {
		return errs.New(
			"The target directory does not appear to contain an ActiveState Runtime installation. Expected to find: %s.",
			installerConfigPath)
	}

	return nil
}

func (r *runner) removeEnvPaths() error {
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}

	// remove shell file additions
	if err := r.shell.CleanUserEnv(r.cfg, sscommon.OfflineInstallID, !isAdmin); err != nil {
		return errs.Wrap(err, "Failed to remove runtime PATH")
	}

	return nil
}
