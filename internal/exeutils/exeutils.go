package exeutils

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
)

// Executables will return all the Executables that need to be symlinked in the various provided bin directories
func Executables(bins []string) ([]string, error) {
	exes := []string{}

	for _, bin := range bins {
		err := filepath.Walk(bin, func(fpath string, info os.FileInfo, err error) error {
			// Filter out files that are not executable
			if info == nil || info.IsDir() || !fileutils.IsExecutable(fpath) { // check if executable by anyone
				return nil // not executable
			}

			exes = append(exes, fpath)
			return nil
		})
		if err != nil {
			return exes, errs.Wrap(err, "Error while walking path")
		}
	}

	return exes, nil
}

type exeFile struct {
	fpath string
	name  string
	ext   string
}

// UniqueExes filters the array of executables for those that would be selected by the command shell in case of a name collision
func UniqueExes(exePaths []string, pathext string) ([]string, error) {
	pathExt := strings.Split(strings.ToLower(pathext), ";")
	exeFiles := map[string]exeFile{}
	result := []string{}

	for _, exePath := range exePaths {
		if runtime.GOOS == "windows" {
			exePath = strings.ToLower(exePath) // Windows is case-insensitive
		}

		exe := exeFile{exePath, "", ""}
		ext := filepath.Ext(exePath)

		// We only set the executable extension if PATHEXT is present.
		// Some macOS builds can contain binaries with periods in their
		// names and we do not want to strip off suffixes after the period.
		if funk.Contains(pathExt, ext) {
			exe.ext = filepath.Ext(exePath)
		}
		exe.name = strings.TrimSuffix(filepath.Base(exePath), exe.ext)

		if prevExe, exists := exeFiles[exe.name]; exists {
			pathsEqual, err := fileutils.PathsEqual(filepath.Dir(exe.fpath), filepath.Dir(prevExe.fpath))
			if err != nil {
				return result, errs.Wrap(err, "Could not compare paths")
			}
			if !pathsEqual {
				continue // Earlier PATH entries win
			}
			if funk.IndexOf(pathExt, prevExe.ext) < funk.IndexOf(pathExt, exe.ext) {
				continue // Earlier PATHEXT entries win
			}
		}

		exeFiles[exe.name] = exe
	}

	for _, exe := range exeFiles {
		result = append(result, exe.fpath)
	}
	return result, nil
}

func ExecSimple(bin string, args ...string) (string, string, error) {
	return ExecSimpleFromDir("", bin, args...)
}

func ExecSimpleFromDir(dir, bin string, args ...string) (string, string, error) {
	c := exec.Command(bin, args...)
	if dir != "" {
		c.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		return stdout.String(), stderr.String(), errs.Wrap(err, "Exec failed")
	}

	return stdout.String(), stderr.String(), nil
}

// Execute will run the given command and with optional settings for the exec.Cmd struct
func Execute(command string, arg []string, optSetter func(cmd *exec.Cmd) error) (int, *exec.Cmd, error) {
	logging.Debug("Executing command: %s, %v", command, arg)

	cmd := exec.Command(command, arg...)

	if optSetter != nil {
		if err := optSetter(cmd); err != nil {
			return -1, nil, err
		}
	}

	err := cmd.Run()
	if err != nil {
		logging.Debug("Executing command returned error: %v", err)
	}
	return osutils.CmdExitCode(cmd), cmd, err
}

// ExecuteAndPipeStd will run the given command and pipe stdin, stdout and stderr
func ExecuteAndPipeStd(command string, arg []string, env []string) (int, *exec.Cmd, error) {
	logging.Debug("Executing command and piping std: %s, %v", command, arg)

	return Execute(command, arg, func(cmd *exec.Cmd) error {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, env...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return nil
	})
}

// ExecuteAndForget will run the given command in the background, returning immediately.
func ExecuteAndForget(command string, args ...string) (*os.Process, error) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = osutils.SysProcAttrForBackgroundProcess()
	if err := cmd.Start(); err != nil {
		return nil, errs.Wrap(err, "Could not start %s %v", command, args)
	}
	cmd.Stdin = nil
	return cmd.Process, nil
}
