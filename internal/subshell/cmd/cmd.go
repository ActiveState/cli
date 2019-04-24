package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
)

var escaper *osutils.ShellEscape

func init() {
	escaper = osutils.NewBatchEscaper()
}

// SubShell covers the subshell.SubShell interface, reference that for documentation
type SubShell struct {
	binary string
	rcFile *os.File
	cmd    *exec.Cmd
	wg     *sync.WaitGroup
	env    []string
}

// Shell - see subshell.SubShell
func (v *SubShell) Shell() string {
	return "cmd"
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
	return ".bat"
}

// RcFileTemplate - see subshell.SubShell
func (v *SubShell) RcFileTemplate() string {
	return "config.bat"
}

// SetEnv - see subshell.SetEnv
func (v *SubShell) SetEnv(env []string) {
	v.env = env
}

// Quote - see subshell.Quote
func (v *SubShell) Quote(value string) string {
	return escaper.Quote(value)
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate(wg *sync.WaitGroup) error {
	v.wg = wg
	wg.Add(1)

	shellArgs := []string{"/K", v.rcFile.Name()}

	cmd := exec.Command("cmd", shellArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()

	v.cmd = cmd

	var err error
	go func() {
		// Intentionally ignore error from command.  Given this is an on going
		// terminal session that the user interacts with, they would have seen
		// any errors already and dealt with them.
		err = cmd.Wait()
		if err != nil {
			logging.Warning(err.Error())
		}
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
	tmpfile, err := ioutil.TempFile("", "batch-script*.bat")
	if err != nil {
		return 1, err
	}

	tmpfile.WriteString(script)
	tmpfile.Close()
	os.Chmod(tmpfile.Name(), 0755)

	runCmd := exec.Command(tmpfile.Name(), args...)
	runCmd.Stdin, runCmd.Stdout, runCmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	runCmd.Env = v.env
	err = runCmd.Run()
	return osutils.CmdExitCode(runCmd), err
}

// IsActive - see subshell.SubShell
func (v *SubShell) IsActive() bool {
	return v.cmd != nil && (v.cmd.ProcessState == nil || !v.cmd.ProcessState.Exited())
}
