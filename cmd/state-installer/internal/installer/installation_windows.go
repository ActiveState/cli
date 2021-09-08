package installer

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

// PrepareBinTargets will move aside any targets in the bin dir that we would otherwise overwrite.
// This guards us from file in use errors as well as false positives by security software
func (i *Installation) PrepareBinTargets() error {
	files, err := ioutil.ReadDir(filepath.Join(i.fromDir, "bin"))
	if err != nil {
		return errs.Wrap(err, "Could not read target dir")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Clean up executables from previous install
		sourceFile := filepath.Join(i.binaryDir, file.Name())
		if strings.HasSuffix(file.Name(), ".old") {
			if err := os.Remove(sourceFile); err != nil {
				logging.Error("Failed to remove old executable: %s. Error: %s", sourceFile, errs.JoinMessage(err))
			}
			continue
		}

		// Move executables aside
		targetFile := filepath.Join(i.binaryDir, fmt.Sprintf("%s-%d.old", file.Name(), time.Now().Unix()))
		if fileutils.TargetExists(sourceFile) {
			if err := os.Rename(sourceFile, targetFile); err != nil {
				return errs.Wrap(err, "Could not move executable aside prior to install: %s to %s", sourceFile, targetFile)
			}
		}
	}

	return nil
}
