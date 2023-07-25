package paygen

import (
	"fmt"
	"os"
	fp "path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
)

var log = func(msg string, vals ...any) {
	fmt.Fprintf(os.Stdout, msg, vals...)
	fmt.Fprintf(os.Stdout, "\n")
}

func GeneratePayload(buildDir, payloadDir string) error {
	emsg := "generate payload: %w"

	payloadBinDir := fp.Join(payloadDir, "bin")

	if err := fileutils.MkdirUnlessExists(payloadBinDir); err != nil {
		return fmt.Errorf(emsg, err)
	}

	installDirMarker := installation.InstallDirMarker
	log("Creating install dir marker in %s", payloadDir)
	if err := fileutils.Touch(fp.Join(payloadDir, installDirMarker)); err != nil {
		return fmt.Errorf(emsg, err)
	}

	files := map[string]string{
		fp.Join(buildDir, constants.StateInstallerCmd+exeutils.Extension): payloadDir,
		fp.Join(buildDir, constants.StateCmd+exeutils.Extension):          payloadBinDir,
		fp.Join(buildDir, constants.StateSvcCmd+exeutils.Extension):       payloadBinDir,
		fp.Join(buildDir, constants.StateExecutorCmd+exeutils.Extension):  payloadBinDir,
	}
	if err := copyFiles(files); err != nil {
		return fmt.Errorf(emsg, err)
	}

	return nil
}

func copyFiles(files map[string]string) error {
	for src, target := range files {
		log("Copying %s to %s", src, target)
		err := fileutils.CopyFile(src, fp.Join(target, fp.Base(src)))
		if err != nil {
			return fmt.Errorf("copy files (item %s to %s): %w", src, target, err)
		}
	}
	return nil
}
