package zsh

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/output"
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
	errs   chan error
}

const Name string = "zsh"

// Shell - see subshell.SubShell
func (v *SubShell) Shell() string {
	return Name
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
func (v *SubShell) WriteUserEnv(cfg sscommon.Configurable, env map[string]string, envType sscommon.RcIdentification, _ bool) error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "RcFile failure")
	}

	env = sscommon.EscapeEnv(env)
	return sscommon.WriteRcFile("zshrc_append.sh", rcFile, envType, env)
}

func (v *SubShell) CleanUserEnv(cfg sscommon.Configurable, envType sscommon.RcIdentification, _ bool) error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "RcFile failure")
	}

	if err := sscommon.CleanRcFile(rcFile, envType); err != nil {
		return errs.Wrap(err, "Failed to remove %s from rcFile", envType)
	}

	return nil
}

func (v *SubShell) RemoveLegacyInstallPath(cfg sscommon.Configurable) error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "RcFile-failure")
	}

	return sscommon.RemoveLegacyInstallPath(rcFile)
}

func (v *SubShell) WriteCompletionScript(completionScript string) error {
	dir := "/usr/local/share/zsh/site-functions"
	if fpath := os.Getenv("FPATH"); fpath != "" {
		fpathv := strings.Split(fpath, ":")
		if len(fpathv) > 0 {
			dir = fpathv[0]
		}
	}

	fpath := filepath.Join(dir, "_"+constants.CommandName)
	err := fileutils.WriteFile(fpath, []byte(completionScript))
	if err != nil {
		logging.Debug("Could not write completions script '%s', likely due to non-admin privileges", fpath)
	}

	homeDir, err := user.HomeDir()
	if err != nil {
		return errs.Wrap(err, "IO failure")
	}

	// Remove the zsh completions cache so our completion script actually gets picked up
	if err := os.Remove(filepath.Join(homeDir, ".zcompdump")); err != nil {
		// non-critical, we're just trying to eliminate any issues caused by zsh's caching
		logging.Debug("Could not delete .zcompdump: %v", err)
	}

	return nil
}

func (v *SubShell) RcFile() (string, error) {
	homeDir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "IO failure")
	}

	return filepath.Join(homeDir, ".zshrc"), nil
}

func (v *SubShell) EnsureRcFileExists() error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "Could not determine rc file")
	}

	return fileutils.TouchFileUnlessExists(rcFile)
}

// SetupShellRcFile - subshell.SubShell
func (v *SubShell) SetupShellRcFile(targetDir string, env map[string]string, namespace *project.Namespaced, cfg sscommon.Configurable) error {
	env = sscommon.EscapeEnv(env)
	return sscommon.SetupShellRcFile(filepath.Join(targetDir, "shell.zsh"), "zshrc_global.sh", env, namespace, cfg)
}

// SetEnv - see subshell.SetEnv
func (v *SubShell) SetEnv(env map[string]string) error {
	v.env = env
	return nil
}

// Quote - see subshell.Quote
func (v *SubShell) Quote(value string) string {
	return escaper.Quote(value)
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate(proj *project.Project, cfg sscommon.Configurable, out output.Outputer) error {
	var directEnv []string

	// available project files require more intensive modification of shell envs
	if proj != nil {
		env := sscommon.EscapeEnv(v.env)
		var err error
		if v.rcFile, err = sscommon.SetupProjectRcFile(proj, "zshrc.sh", "", env, out, cfg, false); err != nil {
			return err
		}

		path, err := os.MkdirTemp("", "state-zsh")
		if err != nil {
			return errs.Wrap(err, "OS failure")
		}

		activeZsrcPath := filepath.Join(path, ".zshrc")
		err = fileutils.CopyFile(v.rcFile.Name(), activeZsrcPath)
		if err != nil {
			return err
		}

		// If users have set $ZDOTDIR then we need to make sure their zshrc file uses it
		// and if it hasn't been set, user $HOME as that is often a default for zsh setup
		// commands.
		userzdotdir := os.Getenv("ZDOTDIR")
		if userzdotdir == "" {
			homeDir, err := user.HomeDir()
			if err != nil {
				log.Println(locale.T("Warning: Could not load home directory for current user"))
			} else {
				userzdotdir = homeDir
			}
		}

		err = fileutils.PrependToFile(activeZsrcPath, []byte(fmt.Sprintf("export ZDOTDIR=%s\n", userzdotdir)))
		if err != nil {
			return err
		}
		os.Setenv("ZDOTDIR", path)
	} else {
		directEnv = sscommon.EnvSlice(v.env)
	}

	cmd := sscommon.NewCommand(v.Binary(), nil, directEnv)
	v.errs = sscommon.Start(cmd)
	v.cmd = cmd
	return nil
}

// Errors returns a channel for receiving errors related to active behavior
func (v *SubShell) Errors() <-chan error {
	return v.errs
}

// Deactivate - see subshell.SubShell
func (v *SubShell) Deactivate() error {
	if !v.IsActive() {
		return nil
	}

	if err := sscommon.Stop(v.cmd); err != nil {
		return err
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

func (v *SubShell) IsAvailable() bool {
	rcFile, err := v.RcFile()
	if err != nil {
		logging.Error("Could not determine rcFile: %s", err)
		return false
	}
	return fileutils.FileExists(rcFile)
}
