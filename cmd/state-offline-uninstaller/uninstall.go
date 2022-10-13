package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	ac "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
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
	}
}

const installerConfigFileName = "installer_config.json"

type InstallerConfig struct {
	OrgName     *string `json:"org_name"`
	ProjectID   *string `json:"project_id"`
	ProjectName *string `json:"project_name"`
	CommitID    *string `json:"commit_id"`
}

type Params struct {
	path string
}

func newParams() *Params {
	return &Params{path: "/tmp"}
}

func (r *runner) Event(eventType string, installerDimensions *dimensions.Values) {
	r.analytics.Event(ac.CatOfflineInstaller, eventType, installerDimensions)
}

func (r *runner) EventWithLabel(eventType string, msg string, installerDimensions *dimensions.Values) {
	r.analytics.EventWithLabel(ac.CatOfflineInstaller, eventType, msg, installerDimensions)
}

func (r *runner) handleFailure(err error, msg string, installerDimensions *dimensions.Values) error {
	r.EventWithLabel(ac.ActOfflineInstallerFailure, msg, installerDimensions)
	return errs.Wrap(err, msg)
}

func (r *runner) handleFailureNewErr(msg string, installerDimensions *dimensions.Values) error {
	r.EventWithLabel(ac.ActOfflineInstallerFailure, msg, installerDimensions)
	return errs.New(msg)
}

func (r *runner) Run(params *Params) error {
	licenseFilePath := filepath.Join(params.path, licenseFileName)
	installerConfigPath := filepath.Join(params.path, installerConfigFileName)

	configData, err := os.ReadFile(installerConfigPath)
	if err != nil {
		return errs.Wrap(err, "Failed to read config file, is this an install directory?")
	}
	config := InstallerConfig{}

	if err := json.Unmarshal([]byte(configData), &config); err != nil {
		return errs.Wrap(err, "Failed to decode config file")
	}

	installerDimensions := &dimensions.Values{
		ProjectNameSpace: p.StrP(project.NewNamespace(*config.OrgName, *config.ProjectName, "").String()),
		CommitID:         config.CommitID,
		Trigger:          p.StrP(target.TriggerOfflineUninstaller.String()),
	}
	r.analytics.Event(ac.CatOfflineInstaller, ac.ActOfflineInstallerStart, installerDimensions)

	containsLicenseFile, err := fileutils.FileContains(licenseFilePath, []byte("ACTIVESTATE"))
	if err != nil {
		return r.handleFailure(err, "Failed to find valid license file, is this an install directory?", installerDimensions)
	}

	if !containsLicenseFile {
		confirmUninstall, err := r.prompt.Confirm(
			"Uninstall",
			"Directory does not look like an install directory, are you sure you want to proceed?",
			p.BoolP(true))
		if err != nil {
			return r.handleFailure(err, "Error getting confirmation for installing", installerDimensions)
		}

		if !confirmUninstall {
			return r.handleFailureNewErr("ActiveState license not found in uninstall directory. Please specify a valid uninstall directory.", installerDimensions)
		}
	}

	r.out.Print("Removing environment configuration")
	err = r.removeEnvPaths()
	if err != nil {
		return r.handleFailure(err, "Error removing environment path", installerDimensions)
	}

	r.out.Print("Removing installation directory")
	err = os.RemoveAll(params.path)
	if err != nil {
		return r.handleFailure(err, "Error removing installation directory", installerDimensions)
	}

	r.analytics.Event(ac.CatOfflineInstaller, ac.ActOfflineInstallerSuccess, installerDimensions)
	r.analytics.Event(ac.CatRuntimeUsage, ac.ActRuntimeDelete, installerDimensions)

	r.out.Print("Uninstall Complete")

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
