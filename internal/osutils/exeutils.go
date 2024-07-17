package osutils

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

func ExecutablePaths(env map[string]string) ([]string, error) {
	// Retrieve artifact binary directory
	var bins []string
	p, ok := env["PATH"]
	if !ok {
		return []string{}, nil
	}
	bins = strings.Split(p, string(os.PathListSeparator))

	exes, err := Executables(bins)
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect executables")
	}

	// Remove duplicate executables as per PATH and PATHEXT
	exes, err = UniqueExes(exes, os.Getenv("PATHEXT"))
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect unique executables, make sure your PATH and PATHEXT environment variables are properly configured.")
	}

	return exes, nil
}

// Executables will find all directories that contain executables from the provided list of paths
func Executables(bins []string) ([]string, error) {
	exes := []string{}

	for _, bin := range bins {
		if !fileutils.DirExists(bin) {
			continue
		}

		entries, err := os.ReadDir(bin)
		if err != nil {
			return nil, errs.Wrap(err, "Could not read directory: %s", bin)
		}

		for _, entry := range entries {
			fpath := filepath.Join(bin, entry.Name())
			if fileutils.IsExecutable(fpath) {
				exes = append(exes, fpath)
			}
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

func ExecSimple(bin string, args []string, env []string) (string, string, error) {
	return ExecSimpleFromDir("", bin, args, env)
}

func ExecSimpleFromDir(dir, bin string, args []string, env []string) (string, string, error) {
	logging.Debug("ExecSimpleFromDir: dir: %s, bin: %s, args: %#v, env: %#v", dir, bin, args, env)
	c := Command(bin, args...)
	if dir != "" {
		c.Dir = dir
	}
	c.Env = os.Environ()
	c.Env = append(c.Env, env...)
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
	cmd := Command(command, arg...)
	logging.Debug("Executing command: %s, with args: %s", cmd, arg)
	if optSetter != nil {
		if err := optSetter(cmd); err != nil {
			return -1, nil, err
		}
	}

	err := cmd.Run()
	if err != nil {
		logging.Debug("Executing command returned error: %v", err)
	}
	return CmdExitCode(cmd), cmd, err
}

// ExecuteAndPipeStd will run the given command and pipe stdin, stdout and stderr
func ExecuteAndPipeStd(command string, arg []string, env []string) (int, *exec.Cmd, error) {
	return Execute(command, arg, func(cmd *exec.Cmd) error {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, env...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return nil
	})
}

// ExecuteAndForget will run the given command in the background, returning immediately.
func ExecuteAndForget(command string, args []string, opts ...func(cmd *exec.Cmd) error) (*os.Process, error) {
	logging.Debug("Executing: %s %v", command, args)
	cmd := Command(command, args...)

	for _, optSetter := range opts {
		if err := optSetter(cmd); err != nil {
			return nil, err
		}
	}

	cmd.SysProcAttr = SysProcAttrForBackgroundProcess()
	if err := cmd.Start(); err != nil {
		return nil, errs.Wrap(err, "Could not start %s %v", command, args)
	}
	cmd.Stdin = nil

	// Wait for the command to finish in a go-routine.  If we do not do that, and the parent process keeps running,
	// the launched process will keep around flagged <defunct> (at least on Linux)
	go func() {
		_ = cmd.Wait()
	}()

	return cmd.Process, nil
}

// ExecuteInBackground runs the command in background and returns a buffers for stdout and stderr
func ExecuteInBackground(command string, args []string, opts ...func(cmd *exec.Cmd) error) (*exec.Cmd, *bytes.Buffer, *bytes.Buffer, error) {
	logging.Debug("Executing: %s %v", command, args)

	cmd := Command(command, args...)
	var stdoutBuf, stderrBuf bytes.Buffer

	for _, optSetter := range opts {
		if err := optSetter(cmd); err != nil {
			return nil, nil, nil, err
		}
	}

	stdoutIn, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, errs.Wrap(err, "Error creating StdoutPipe for Cmd")
	}

	stderrIn, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, errs.Wrap(err, "Error creating StderrPipe for Cmd")
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, errs.Wrap(err, "Could not start %s %v", command, args)
	}

	// Copy the output in a separate goroutine so printing can't block indefinitely.
	// We use a WaitGroup to wait for the command to exit and for the goroutine to return.
	go func() {
		_, _ = io.Copy(&stdoutBuf, stdoutIn)
		_, _ = io.Copy(&stderrBuf, stderrIn)
	}()

	// Return a function that can be called to get the output so far
	return cmd, &stdoutBuf, &stderrBuf, nil
}

// DecodeCmd takes an encoded command and decodes it by returning a shell variant based on the OS we're on
func DecodeCmd(cmd string) (string, []string) {
	switch runtime.GOOS {
	case "windows":
		return "cmd", []string{"/C", cmd}
	default:
		return "sh", []string{"-c", cmd}
	}
}
