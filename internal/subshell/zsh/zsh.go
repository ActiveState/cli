package zsh

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
)

var escaper *osutils.ShellEscape

func init() {
	escaper = osutils.NewBashEscaper()
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
	return "zsh"
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
	return "zshrc.sh"
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

	path, err := ioutil.TempDir("", "state-zsh")
	if err != nil {
		return err
	}

	activeZsrcPath := filepath.Join(path, ".zshrc")
	fail := fileutils.CopyFile(v.rcFile.Name(), activeZsrcPath)
	if fail != nil {
		return fail
	}

	// If users have set $ZDOTDIR then we need to make sure their zshrc file uses it
	// and if it hasn't been set, user $HOME as that is often a default for zsh setup
	// commands.
	userzdotdir := os.Getenv("ZDOTDIR")
	if userzdotdir == "" {
		u, err := user.Current()
		if err != nil {
			log.Println(locale.T("Warning: Could not load home directory for current user"))
		} else {
			userzdotdir = u.HomeDir
		}
	}

	fail = fileutils.PrependToFile(activeZsrcPath, []byte(fmt.Sprintf("export ZDOTDIR=%s\n", userzdotdir)))
	if fail != nil {
		return fail
	}
	os.Setenv("ZDOTDIR", path)

	shellArgs := []string{}
	cmd := exec.Command(v.Binary(), shellArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()

	v.cmd = cmd

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
	tmpfile, err := ioutil.TempFile("", "bash-script")
	if err != nil {
		return 1, err
	}

	tmpfile.WriteString("#!/usr/bin/env bash\n")
	tmpfile.WriteString(script)
	tmpfile.Close()
	os.Chmod(tmpfile.Name(), 0755)

	filePath, fail := osutils.BashifyPath(tmpfile.Name())
	if fail != nil {
		return 1, fail.ToError()
	}

	quotedArgs := []string{filePath}
	for _, arg := range args {
		quotedArgs = append(quotedArgs, v.Quote(arg))
	}

	runCmd := exec.Command(v.Binary(), "-c", strings.Join(quotedArgs, " "))
	runCmd.Stdin, runCmd.Stdout, runCmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	runCmd.Env = v.env

	err = runCmd.Run()
	return osutils.CmdExitCode(runCmd), err
}

// IsActive - see subshell.SubShell
func (v *SubShell) IsActive() bool {
	return v.cmd != nil && (v.cmd.ProcessState == nil || !v.cmd.ProcessState.Exited())
}
