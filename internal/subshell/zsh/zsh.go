package zsh

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/project"
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
	env    map[string]string
	fs     chan *failures.Failure
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

// WriteUserEnv - see subshell.SubShell
func (v *SubShell) WriteUserEnv(env map[string]string, _ bool) *failures.Failure {
	homeDir, err := fileutils.HomeDir()
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	env = sscommon.EscapeEnv(env)
	return sscommon.WriteRcFile("zshrc_append.sh", filepath.Join(homeDir, ".zshrc"), env)
}

// SetupShellRcFile - subshell.SubShell
func (v *SubShell) SetupShellRcFile(targetDir string, env map[string]string, namespace project.Namespaced) error {
	env = sscommon.EscapeEnv(env)
	return sscommon.SetupShellRcFile(filepath.Join(targetDir, "shell.zsh"), "zshrc.sh", env, namespace)
}

// SetEnv - see subshell.SetEnv
func (v *SubShell) SetEnv(env map[string]string) {
	v.env = env
}

// Quote - see subshell.Quote
func (v *SubShell) Quote(value string) string {
	return escaper.Quote(value)
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate() *failures.Failure {
	env := sscommon.EscapeEnv(v.env)
	var fail *failures.Failure
	if v.rcFile, fail = sscommon.SetupProjectRcFile("zshrc.sh", "", env); fail != nil {
		return fail
	}

	path, err := ioutil.TempDir("", "state-zsh")
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	activeZsrcPath := filepath.Join(path, ".zshrc")
	fail = fileutils.CopyFile(v.rcFile.Name(), activeZsrcPath)
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

	v.fs = sscommon.Start(cmd)
	v.cmd = cmd
	return nil
}

// Failures returns a channel for receiving errors related to active behavior
func (v *SubShell) Failures() <-chan *failures.Failure {
	return v.fs
}

// Deactivate - see subshell.SubShell
func (v *SubShell) Deactivate() *failures.Failure {
	if !v.IsActive() {
		return nil
	}

	if fail := sscommon.Stop(v.cmd); fail != nil {
		return fail
	}

	v.cmd = nil
	return nil
}

// Run - see subshell.SubShell
func (v *SubShell) Run(filename string, args ...string) error {
	return sscommon.RunFuncByBinary(v.Binary())(osutils.EnvMapToSlice(v.env), filename, args...)
}

// IsActive - see subshell.SubShell
func (v *SubShell) IsActive() bool {
	return v.cmd != nil && (v.cmd.ProcessState == nil || !v.cmd.ProcessState.Exited())
}
