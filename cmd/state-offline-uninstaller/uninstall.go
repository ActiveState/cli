package main

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/p"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
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

type Params struct {
	path string
}

func newParams() *Params {
	return &Params{path: "/tmp"}
}

func (r *runner) Run(params *Params) error {
	licenseFilePath := filepath.Join(params.path, licenseFileName)
	containsLicenseFile, err := fileutils.FileContains(licenseFilePath, []byte("ACTIVESTATE"))
	if err != nil {
		return errs.Wrap(err, "Error determining if directory is an install directory")
	}

	if !containsLicenseFile {
		confirmUninstall, err := r.prompt.Confirm(
			"Uninstall",
			"Directory does not look like an install directory, are you sure you want to proceed?",
			p.BoolP(true))
		if err != nil {
			return errs.Wrap(err, "Error getting confirmation for installing")
		}

		if !confirmUninstall {
			return errs.New("ActiveState license not found in uninstall directory. Please specify a valid uninstall directory.")
		}
	}

	r.out.Print("Removing environment configuration")
	err = r.removeEnvPaths()
	if err != nil {
		return errs.Wrap(err, "Error removing environment path")
	}

	r.out.Print("Removing installation directory")
	err = os.RemoveAll(params.path)
	if err != nil {
		return errs.Wrap(err, "Error removing installation directory")
	}

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
