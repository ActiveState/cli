package osutils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
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

// CmdString returns a human-readable description of c.
// This is a copy of the Go 1.13 (cmd.String) function
func CmdString(c *exec.Cmd) string {

	// report the exact executable path (plus args)
	b := new(strings.Builder)
	b.WriteString(c.Path)

	for _, a := range c.Args[1:] {
		b.WriteByte(' ')
		b.WriteString(a)
	}

	return b.String()
}

// BashifyPath takes a windows style path and turns it into a bash style path
// eg. C:\temp becomes /c/temp
func BashifyPath(absolutePath string) (string, error) {
	if absolutePath[0:1] == "/" {
		// Already the format we want
		return absolutePath, nil
	}

	if absolutePath[1:2] != ":" {
		// Check for windows style paths
		return "", errs.New("Unrecognized absolute path format: %s", absolutePath)
	}

	absolutePath = strings.ToLower(absolutePath[0:1]) + absolutePath[2:]
	absolutePath = strings.Replace(absolutePath, `\`, `/`, -1)  // backslash to forward slash
	absolutePath = strings.Replace(absolutePath, ` `, `\ `, -1) // escape space
	return "/" + absolutePath, nil
}

// Getwd is an alias of osutils.Getwd which wraps the error in our localized error message and FailGetWd, which is user facing (doesn't get logged)
func Getwd() (string, error) {
	r, err := os.Getwd()
	if err != nil {
		return "", errs.Wrap(err, "GetWd failed")
	}
	return r, nil
}

func EnvSliceToMap(envSlice []string) map[string]string {
	env := map[string]string{}
	for _, v := range envSlice {
		kv := strings.SplitN(v, "=", 2)
		env[kv[0]] = ""
		if len(kv) == 2 { // account for empty values, windows does some weird stuff, better safe than sorry
			env[kv[0]] = kv[1]
		}
	}
	return env
}

func EnvMapToSlice(envMap map[string]string) []string {
	var env []string
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// Executable returns the resolved path to the currently running executable.
func Executable() (string, error) {
	exec, err := os.Executable()
	if err != nil {
		return "", err
	}

	return fileutils.ResolvePath(exec)
}
