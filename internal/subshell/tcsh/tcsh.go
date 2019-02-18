package tcsh

import (
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"github.com/ActiveState/cli/internal/osutils"

	"github.com/ActiveState/cli/internal/failures"
)

// SubShell covers the subshell.SubShell interface, reference that for documentation
type SubShell struct {
	binary string
	rcFile *os.File
	cmd    *exec.Cmd
	wg     *sync.WaitGroup
}

// Shell - see subshell.SubShell
func (v *SubShell) Shell() string {
	return "tcsh"
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
func (v *SubShell) SetRcFile(rcFile *os.File) {
	v.rcFile = rcFile
}

// RcFileExt - see subshell.SubShell
func (v *SubShell) RcFileExt() string {
	return ""
}

// RcFileTemplate - see subshell.SubShell
func (v *SubShell) RcFileTemplate() string {
	return "tcsh.sh"
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate(wg *sync.WaitGroup) error {
	v.wg = wg
	wg.Add(1)

	// This is horrible but it works.  tcsh doesn't offer a way to override the rc file and
	// doesn't let us run a script and then drop to interactive mode.  So we source the
	// state rc file and then chain an exec which inherits the environment we just set up.
	// It seems to work fine except we don't have a way to override the shell prompt.
	//
	// The exec'd shell does not inherit 'prompt' from the calling terminal since
	// tcsh does not export prompt.  This may be intractable.  I couldn't figure out a
	// hack to make it work.
	shellArgs := []string{"-c", "source " + v.rcFile.Name() + " ; exec " + v.Binary()}
	cmd := exec.Command(v.Binary(), shellArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()

	v.cmd = cmd

	var err error
	go func() {
		err = cmd.Wait()
		v.wg.Done()
	}()

	return err
}

// Deactivate - see subshell.SubShell
func (v *SubShell) Deactivate() error {
	if !v.IsActive() {
		return nil
	}

	var err error
	func() {
		// Go's Process.Kill is not very safe to use, it throws a panic if the process no longer exists
		defer failures.Recover()
		err = v.cmd.Process.Kill()
	}()

	if err == nil {
		v.cmd = nil
	}
	return err
}

// Run - see subshell.SubShell
func (v *SubShell) Run(script string, args ...string) (int, error) {
	tmpfile, err := ioutil.TempFile("", "tcsh-script")
	if err != nil {
		return 1, err
	}

	tmpfile.WriteString("#!/usr/bin/env tcsh\n")
	tmpfile.WriteString(script)
	tmpfile.Close()
	os.Chmod(tmpfile.Name(), 0755)

	runCmd := exec.Command(tmpfile.Name(), args...)
	runCmd.Stdin, runCmd.Stdout, runCmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	err = runCmd.Run()
	return osutils.CmdExitCode(runCmd), err
}

// IsActive - see subshell.SubShell
func (v *SubShell) IsActive() bool {
	return v.cmd != nil && (v.cmd.ProcessState == nil || !v.cmd.ProcessState.Exited())
}
