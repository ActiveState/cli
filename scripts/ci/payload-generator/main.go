package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
)

// The payload-generator is an intentionally very dumb runner that just copies some files around.
// This could just be a bash script if not for the fact that bash can't be linked with our type system.

func main() {
	root := environment.GetRootPathUnsafe()
	buildDir := filepath.Join(root, "build")
	payloadDir := filepath.Join(buildDir, "payload")
	payloadBinDir := filepath.Join(buildDir, "payload", "bin")

	if err := fileutils.MkdirUnlessExists(payloadBinDir); err != nil {
		fmt.Printf("Error creating payload bin dir: %s\n", err.Error())
		os.Exit(1)
	}

	if err := copyFiles(map[string]string{
		filepath.Join(buildDir, constants.StateInstallerCmd+exeutils.Extension): payloadDir,
		filepath.Join(buildDir, constants.StateCmd+exeutils.Extension):          payloadBinDir,
		filepath.Join(buildDir, constants.StateSvcCmd+exeutils.Extension):       payloadBinDir,
		filepath.Join(buildDir, constants.StateExecutorCmd+exeutils.Extension):  payloadBinDir,
	}); err != nil {
		fmt.Printf("Error copying files: %s\n", err.Error())
		os.Exit(1)
	}

}

func copyFiles(files map[string]string) error {
	for src, target := range files {
		fmt.Printf("Copying %s to %s\n", src, target)
		err := fileutils.CopyFile(src, filepath.Join(target, filepath.Base(src)))
		if err != nil {
			return err
		}
	}
	return nil
}
