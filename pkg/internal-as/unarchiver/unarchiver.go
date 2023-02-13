// Package unarchiver exposes some of the unarchiver internal package.
package unarchiver

import "github.com/ActiveState/cli/internal-as/unarchiver"

func NewTarGz() unarchiver.Unarchiver {
	return unarchiver.NewTarGz()
}

func NewZip() unarchiver.Unarchiver {
	return unarchiver.NewZip()
}
