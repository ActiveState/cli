package main

import (
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"

	"github.com/ActiveState/cli/internal/analytics/client/blackhole"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

// NOTE: From: internal/runners/clean/uninstall.go

func runUnInstall(out output.Outputer, params *Params) error {
	analytics := blackhole.New()
	prompter := prompt.New(true, analytics)
	default_boolean_answer := true
	installToDir := params.path

	if !isInstallDirectory(installToDir) {
		confirmUninstall, err := prompter.Confirm("Uninstall", "Directory does not look like an install directory, are you sure you want to proceed?", &default_boolean_answer)
		if err != nil {
			return errs.Wrap(err, "Error getting confirmation for installing")
		}

		if !confirmUninstall {
			return errs.New("ActiveState license not found in uninstall directory. Please specify a valid uninstall directory.")
		}

	}

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Error creating config")
	}

	out.Print("Removing environment configuration")
	err = removeEnvPaths(cfg)
	if err != nil {
		return errs.Wrap(err, "Error removing environment path")
	}

	out.Print("Removing installation directory")
	err = os.RemoveAll(installToDir)
	if err != nil {
		return errs.Wrap(err, "Error removing installation directory")
	}

	out.Print("Uninstall Complete")

	return nil
}

func removeEnvPaths(cfg sscommon.Configurable) error {
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}

	// remove shell file additions
	s := subshell.New(cfg)
	if err := s.CleanUserEnv(cfg, sscommon.DeployID, !isAdmin); err != nil {
		return errs.Wrap(err, "Failed to remove runtime PATH")
	}

	return nil
}
