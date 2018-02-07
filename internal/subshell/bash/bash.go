package bash

import (
	"os"
	"os/exec"
)

// SubShell covers the subshell.SubShell interface, reference that for documentation
type SubShell struct {
	binary string
	rcFile *os.File
	cmd    *exec.Cmd
}

// Shell - see subshell.SubShell
func (v *SubShell) Shell() string {
	return "bash"
}

// ShellScript - see subshell.SubShell
func (v *SubShell) ShellScript() string {
	return "bashrc.sh"
}

// Binary - see subshell.SubShell
func (v *SubShell) Binary() string {
	return v.binary
}

// SetBinary - see subshell.SubShell
func (v *SubShell) SetBinary(binary string) {
	v.binary = binary
}

// RcFile - see subshell.SubShell
func (v *SubShell) RcFile() *os.File {
	return v.rcFile
}

// SetRcFile - see subshell.SubShell
func (v *SubShell) SetRcFile(rcFile os.File) {
	v.rcFile = &rcFile
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate() error {
	shellArgs := []string{"--rcfile", v.rcFile.Name()}
	cmd := exec.Command(v.Binary(), shellArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()

	v.cmd = cmd

	var err error
	go func() {
		err = cmd.Wait()
	}()

	return err
}

// Deactivate - see subshell.SubShell
func (v *SubShell) Deactivate() error {
	if !v.IsActive() {
		return nil
	}
	err := v.cmd.Process.Kill()
	if err == nil {
		v.cmd = nil
	}
	return err
}

// IsActive - see subshell.SubShell
func (v *SubShell) IsActive() bool {
	return v.cmd != nil && (v.cmd.ProcessState == nil || !v.cmd.ProcessState.Exited())
}
