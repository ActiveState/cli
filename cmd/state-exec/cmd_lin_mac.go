//go:build !windows
// +build !windows

package main

import "os/exec"

func Command(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
