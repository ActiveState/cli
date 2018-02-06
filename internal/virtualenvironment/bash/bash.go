package bash

import (
	"os"
	"os/exec"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	binary string
	rcFile *os.File
	cmd    *exec.Cmd
}

// Shell - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Shell() string {
	return "bash"
}

// ShellScript - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) ShellScript() string {
	return "bashrc.sh"
}

// Binary - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Binary() string {
	return v.binary
}

// SetBinary - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) SetBinary(binary string) {
	v.binary = binary
}

// RcFile - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) RcFile() *os.File {
	return v.rcFile
}

// SetRcFile - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) SetRcFile(rcFile os.File) {
	v.rcFile = &rcFile
}

// Activate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Activate() error {
	shellArgs := []string{"--rcfile", v.rcFile.Name()}
	cmd := exec.Command("bash", shellArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()

	v.cmd = cmd

	var err error
	go func() {
		err = cmd.Wait()
	}()

	return err
}

// Deactivate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Deactivate() error {
	if !v.IsActive() {
		return nil
	}
	err := v.cmd.Process.Kill()
	if err == nil {
		v.cmd = nil
	}
	return err
}

// IsActive - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) IsActive() bool {
	return v.cmd != nil && (v.cmd.ProcessState == nil || !v.cmd.ProcessState.Exited())
}
