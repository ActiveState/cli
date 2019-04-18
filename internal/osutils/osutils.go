package osutils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/failures"

	"github.com/ActiveState/cli/internal/logging"
)

// CmdExitCode returns the exit code of a command in a platform agnostic way
// taken from https://www.reddit.com/r/golang/comments/1hvvnn/any_better_way_to_do_a_crossplatform_exec_and/caytqvr/
func CmdExitCode(cmd *exec.Cmd) (code int) {
	defer func() {
		if r := recover(); r != nil {
			logging.Errorf("Could not get exit code, so returning 1 instead (this is non-fatal, but should be resolved), actual error: %v", r)
			code = 128
		}
	}()

	type Status interface {
		ExitStatus() int
	}
	return cmd.ProcessState.Sys().(Status).ExitStatus()
}

// ExecuteAndPipeStd will run the given command and pipe stdin, stdout and stderr
func ExecuteAndPipeStd(command string, arg []string, env []string) (int, *exec.Cmd, error) {
	logging.Debug("Executing command and piping std: %s, %v", command, arg)

	cmd := exec.Command(command, arg...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	err := cmd.Run()
	if err != nil {
		logging.Error("Executing command returned error: %v", err)
	}
	return CmdExitCode(cmd), cmd, err
}

// ShellEscapeAndQuote will escape and quote the given strings
func ShellEscapeAndQuote(strs ...string) []string {
	result := []string{}
	for _, str := range strs {
		str = strconv.Quote(str)
		if runtime.GOOS == "windows" && os.Getenv("SHELL") == "" {
			str = strings.Replace(str, "\\\\", "\\", -1)
		}
		result = append(result, str)
	}
	return result
}

// BashifyPath takes a windows style path and turns it into a bash style path
// eg. C:\temp becomes /c/temp
func BashifyPath(path string) (string, *failures.Failure) {
	if path[0:1] == "/" {
		// Already the format we want
		return path, nil
	}

	if path[1:2] != ":" {
		// Check for windows style paths
		return "", failures.FailInput.New(fmt.Sprintf("Unrecognized path format: %s", path))
	}

	path = strings.ToLower(path[0:1]) + path[2:]
	return "/" + strings.Replace(path, `\`, `/`, -1), nil
}
