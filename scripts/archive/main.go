package main

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	wc "github.com/ActiveState/cli/scripts/internal/workflow-controllers"
	"github.com/google/uuid"
)

func main() {
	if err := run(); err != nil {
		wc.Print("Error: %s\n", errs.JoinMessage(err))
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errs.New("Usage: archive <path-to-archive> <archive-path>")
	}

	root := environment.GetRootPathUnsafe()

	buildPath := filepath.Join(root, "build", "offline")
	artifactPath := filepath.Join(buildPath, uuid.New().String()+".tar.gz")

	archiver := archiver.NewTarGz()
	err := archiver.Archive(fileutils.ListFilesUnsafe(os.Args[1]), artifactPath)
	if err != nil {
		return errs.Wrap(err, "Could not archive files1")
	}

	err = archiver.Archive([]string{artifactPath}, filepath.Join(buildPath, "artifacts.tar.gz"))
	if err != nil {
		return errs.Wrap(err, "Could not archive files2")
	}

	err = os.Remove(artifactPath)
	if err != nil {
		return errs.Wrap(err, "Could not remove archive")
	}

	return nil
}
