// +build windows

package runtime

import (
	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/unarchiver"
)

// InstallerExtension is used to identify whether an artifact is one that we should care about
const InstallerExtension = ".zip"

// Archiver returns the archiver to use
func Archiver() archiver.Archiver {
	return archiver.DefaultZip
}

// UnarchiverWithProgress returns the ProgressUnarchiver to use
func UnarchiverWithProgress() unarchiver.Unarchiver {
	return unarchiver.NewZip()
}
