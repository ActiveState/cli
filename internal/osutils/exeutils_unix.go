//go:build !windows
// +build !windows

package osutils

import "os/exec"

const ExeExtension = ""

var exts = []string{}

func Command(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
