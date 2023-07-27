package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
)

var (
	log = func(msg string, vals ...any) {
		fmt.Fprintf(os.Stdout, msg, vals...)
		fmt.Fprintf(os.Stdout, "\n")
	}
	logErr = func(msg string, vals ...any) {
		fmt.Fprintf(os.Stderr, msg, vals...)
		fmt.Fprintf(os.Stderr, "\n")
	}
)

// The payload-generator is an intentionally very dumb runner that just copies some files around.
// This could just be a bash script if not for the fact that bash can't be linked with our type system.
func main() {
	if err := run(); err != nil {
		logErr("%s", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		branch  = constants.BranchName
		version = constants.Version
	)

	flag.StringVar(&branch, "b", branch, "Override target branch. (Branch to receive update.)")
	flag.StringVar(&version, "v", version, "Override version number for this update.")
	flag.Parse()

	root := environment.GetRootPathUnsafe()
	buildDir := filepath.Join(root, "build")
	payloadDir := filepath.Join(buildDir, "payload")

	return generatePayload(buildDir, payloadDir, branch, version)
}

func generatePayload(buildDir, payloadDir, branch, version string) error {
	emsg := "generate payload: %w"

	payloadBinDir := filepath.Join(payloadDir, "bin")

	if err := fileutils.MkdirUnlessExists(payloadBinDir); err != nil {
		return fmt.Errorf(emsg, err)
	}

	log("Creating install dir marker in %s", payloadDir)
	if err := createInstallMarker(payloadDir, branch, version); err != nil {
		return fmt.Errorf(emsg, err)
	}

	files := map[string]string{
		filepath.Join(buildDir, constants.StateInstallerCmd+exeutils.Extension): payloadDir,
		filepath.Join(buildDir, constants.StateCmd+exeutils.Extension):          payloadBinDir,
		filepath.Join(buildDir, constants.StateSvcCmd+exeutils.Extension):       payloadBinDir,
		filepath.Join(buildDir, constants.StateExecutorCmd+exeutils.Extension):  payloadBinDir,
	}
	if err := copyFiles(files); err != nil {
		return fmt.Errorf(emsg, err)
	}

	return nil
}

func createInstallMarker(payloadDir, branch, version string) error {
	emsg := "create install marker: %w"

	markerContents := installation.InstallMarkerMeta{
		Branch:  branch,
		Version: version,
	}
	b, err := json.Marshal(markerContents)
	if err != nil {
		return fmt.Errorf(emsg, err)
	}

	markerPath := filepath.Join(payloadDir, installation.InstallDirMarker)
	if err := fileutils.WriteFile(markerPath, b); err != nil {
		return fmt.Errorf(emsg, err)
	}

	return nil
}

// copyFiles will copy the given files while preserving permissions.
func copyFiles(files map[string]string) error {
	emsg := "copy files (%s to %s): %w"

	for src, target := range files {
		log("Copying %s to %s", src, target)
		dest := filepath.Join(target, filepath.Base(src))
		err := fileutils.CopyFile(src, dest)
		if err != nil {
			return fmt.Errorf(emsg, src, target, err)
		}
		srcStat, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf(emsg, src, target, err)
		}

		if err := os.Chmod(dest, srcStat.Mode().Perm()); err != nil {
			return fmt.Errorf(emsg, src, target, err)
		}
	}

	return nil
}
