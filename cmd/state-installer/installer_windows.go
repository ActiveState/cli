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
	"github.com/ActiveState/cli/internal/logging"
)

func InstallSystemFiles(_, _, _ string) error {
	return nil
}

func (i *Installer) installLauncher() error {
	return nil
}

// PrepareBinTargets will move aside any targets in the bin dir that we would otherwise overwrite.
// This guards us from file in use errors as well as false positives by security software
func (i *Installer) PrepareBinTargets(useBinDir bool) error {
	sourceBinPath := filepath.Join(i.sourcePath, "bin")
	targetBinPath := i.path

	if useBinDir {
		targetBinPath := filepath.Join(targetBinPath, "bin")
	}

	// Clean up executables from previous install
	if fileutils.DirExists(targetBinPath) {
		files, err := ioutil.ReadDir(targetBinPath)
		if err != nil {
			return errs.Wrap(err, "Could not read target dir")
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if strings.HasSuffix(file.Name(), ".old") {
				logging.Debug("Deleting old file: %s", file.Name())
				oldFile := filepath.Join(targetBinPath, file.Name())
				if err := os.Remove(oldFile); err != nil {
					logging.Error("Failed to remove old executable: %s. Error: %s", oldFile, errs.JoinMessage(err))
				}
				continue
			}
		}
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
