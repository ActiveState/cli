package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
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
	sourceBinPath := filepath.Join(i.sourcePath, "bin")
	targetBinPath := filepath.Join(i.path, "bin")

	// Clean up exectuables from potentially corrupted installed
	err := removeOldExecutables(i.path)
	if err != nil {
		return errs.Wrap(err, "Could not remove executables at: %s", i.path)
	}

	// Clean up exectuables from old install
	err = removeOldExecutables(targetBinPath)
	if err != nil {
		return errs.Wrap(err, "Could not remove executables at: %s", i.path)
	}

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
				logging.Debug("Moving aside conflicting file: %s", targetFile)
				renamedFile := filepath.Join(targetBinPath, fmt.Sprintf("%s-%d.old", file.Name(), time.Now().Unix()))
				if err := os.Rename(targetFile, renamedFile); err != nil {
					return errs.Wrap(err, "Could not move executable aside prior to install: %s to %s", targetFile, renamedFile)
				}
				// Make an attempt to remove the file. This has a decent chance of failing if we're updating.
				// That's just a limitation on Windows and worse case scenario we'll clean it up the next update attempt.
				os.Remove(renamedFile)
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
			renamedFile := filepath.Join(i.path, fmt.Sprintf("%s-%d.old", fname, time.Now().Unix()))
			if err := os.Rename(targetFile, renamedFile); err != nil {
				return errs.Wrap(err, "Could not rename corrupted executable: %s to %s", targetFile, renamedFile)
			}
			// This will likely fail but we try anyways
			os.Remove(renamedFile)
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

func removeOldExecutables(dir string) error {
	if !fileutils.TargetExists(dir) {
		return nil
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return errs.Wrap(err, "Could not read installer dir")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), ".old") {
			logging.Debug("Deleting old file: %s", file.Name())
			oldFile := filepath.Join(dir, file.Name())
			if err := os.Remove(oldFile); err != nil {
				multilog.Error("Failed to remove old executable: %s. Error: %s", oldFile, errs.JoinMessage(err))
			}
		}
	}

	return nil
}
