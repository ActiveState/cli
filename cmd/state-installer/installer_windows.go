package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

func InstallSystemFiles(_, _, _ string) error {
	return nil
}

func (i *Installer) installLauncher() error {
	return nil
}

// PrepareBinTargets will move aside any targets in the bin dir that we would otherwise overwrite.
// This guards us from file in use errors as well as false positives by security software
func (i *Installer) PrepareBinTargets() error {
	sourceBinPath := filepath.Join(i.payloadPath, "bin")
	targetBinPath := filepath.Join(i.path, "bin")

	// Move aside conflicting executables in target
	if fileutils.DirExists(sourceBinPath) {
		files, err := ioutil.ReadDir(sourceBinPath)
		if err != nil {
			return errs.Wrap(err, "Could not read target dir")
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Move executables aside
			targetFile := filepath.Join(targetBinPath, file.Name())
			if fileutils.TargetExists(targetFile) {
				logging.Debug("Attempting to delete conflicting file: %s", targetFile)
				if err := fileutils.DeleteNowOrLater(targetFile); err != nil {
					return errs.Wrap(err, "Could not delete file: %s", targetFile)
				}
			}
		}
	}

	return nil
}

func (i *Installer) sanitizeInstallPath() error {
	if !fileutils.DirExists(i.path) {
		return nil
	}

	files, err := ioutil.ReadDir(i.path)
	if err != nil {
		return errs.Wrap(err, "Could not installation directory: %s", i.path)
	}

	for _, file := range files {
		fname := strings.ToLower(file.Name())
		targetFile := filepath.Join(i.path, file.Name())
		if isStateExecutable(fname) {
			if err := fileutils.DeleteNowOrLater(targetFile); err != nil {
				return errs.Wrap(err, "Could not delete corrupted executable: %s to %s", targetFile)
			}
		}
	}

	installContext, err := installation.GetContext()
	if err != nil {
		return errs.Wrap(err, "Could not get initial installation context")
	}

	// Since we are repairing a corrupted install we need to also remove the old
	// PATH entry. The new PATH entry will be added later in the install/update process.
	// This is only an issue on Windows as on other platforms we can simply rewrite
	// the PATH entry.
	s := subshell.New(i.cfg)
	if err := s.CleanUserEnv(i.cfg, sscommon.InstallID, !installContext.InstalledAsAdmin); err != nil {
		return errs.Wrap(err, "Failed to State Tool installation PATH")
	}

	return nil
}
