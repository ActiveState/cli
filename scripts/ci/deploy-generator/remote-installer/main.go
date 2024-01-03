package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s error: %v", os.Args[0], errs.JoinMessage(err))
	}
}

func run() error {
	version := constants.RemoteInstallerVersion

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if len(os.Args) == 3 {
		goos = os.Args[1]
		goarch = os.Args[2]
	}
	platform := goos + "-" + goarch

	relPath := filepath.Join("remote-installer", constants.ChannelName, platform)
	relVersionedPath := filepath.Join("remote-installer", constants.ChannelName, version, platform)

	buildPath := filepath.Join(environment.GetRootPathUnsafe(), "build")

	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	sourceFile := filepath.Join(buildPath, constants.StateRemoteInstallerCmd+ext)
	if !fileutils.FileExists(sourceFile) {
		return errs.New("source file does not exist: %s", sourceFile)
	}

	fmt.Printf("Copying %s to %s\n", sourceFile, relPath)
	if err := fileutils.CopyFile(sourceFile, filepath.Join(buildPath, relPath, constants.StateRemoteInstallerCmd+ext)); err != nil {
		return errs.Wrap(err, "failed to copy source file to channel path")
	}

	fmt.Printf("Copying %s to %s\n", sourceFile, relVersionedPath)
	if err := fileutils.CopyFile(sourceFile, filepath.Join(buildPath, relVersionedPath, constants.StateRemoteInstallerCmd+ext)); err != nil {
		return errs.Wrap(err, "failed to copy source file to version path")
	}

	return nil
}
