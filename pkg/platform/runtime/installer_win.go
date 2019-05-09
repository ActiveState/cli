// +build windows

package runtime

import "github.com/ActiveState/archiver"

// Archiver returns the archiver to use
func Archiver() archiver.Archiver {
	return archiver.DefaultZip
}

// Unarchiver returns the unarchiver to use
func Unarchiver() archiver.Unarchiver {
	return archiver.DefaultZip
}
