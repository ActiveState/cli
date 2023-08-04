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
	defaultInputDir  = filepath.Join(environment.GetRootPathUnsafe(), "build")
	defaultOutputDir = filepath.Join(defaultInputDir, "payload", "state-install")

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
		inDir   = defaultInputDir
		outDir  = defaultOutputDir
		branch  = constants.BranchName
		version = constants.Version
	)

	flag.StringVar(&branch, "b", branch, "Override target branch. (Branch to receive update.)")
	flag.StringVar(&version, "v", version, "Override version number for this update.")
	flag.Parse()

	return generatePayload(inDir, outDir, branch, version)
}

func generatePayload(inDir, outDir, branch, version string) error {
	emsg := "generate payload: %w"

	binDir := filepath.Join(outDir, "bin")

	if err := fileutils.MkdirUnlessExists(binDir); err != nil {
		return fmt.Errorf(emsg, err)
	}

	log("Creating install dir marker in %s", outDir)
	if err := createInstallMarker(outDir, branch, version); err != nil {
		return fmt.Errorf(emsg, err)
	}

	files := map[string]string{
		filepath.Join(inDir, constants.StateInstallerCmd+exeutils.Extension): outDir,
		filepath.Join(inDir, constants.StateCmd+exeutils.Extension):          binDir,
		filepath.Join(inDir, constants.StateSvcCmd+exeutils.Extension):       binDir,
		filepath.Join(inDir, constants.StateExecutorCmd+exeutils.Extension):  binDir,
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
	b = append(b, '\n')

	markerPath := filepath.Join(payloadDir, installation.InstallDirMarker)
	if err := fileutils.WriteFile(markerPath, b); err != nil {
		return fmt.Errorf(emsg, err)
	}

	return nil
}

// copyFiles will copy the given files with logging.
func copyFiles(files map[string]string) error {
	for src, target := range files {
		log("Copying %s to %s", src, target)
		dest := filepath.Join(target, filepath.Base(src))

		if err := fileutils.CopyFile(src, dest); err != nil {
			return fmt.Errorf("copy files (%s to %s): %w", src, target, err)
		}
	}

	return nil
}
