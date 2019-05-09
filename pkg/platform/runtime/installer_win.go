// +build windows

package runtime

import "github.com/ActiveState/archiver"

// InstallerExtension is used to identify whether an artifact is one that we should care about
const InstallerExtension = ".zip"

// Archiver returns the archiver to use
func Archiver() archiver.Archiver {
	return archiver.DefaultZip
}

// Unarchiver returns the unarchiver to use
func Unarchiver() archiver.Unarchiver {
	return archiver.DefaultZip
}
