package osutils

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rollbar"
	"golang.org/x/sys/windows"
)

// CmdExitCode returns the exit code of a command in a platform agnostic way
// taken from https://www.reddit.com/r/golang/comments/1hvvnn/any_better_way_to_do_a_crossplatform_exec_and/caytqvr/
func CmdExitCode(cmd *exec.Cmd) (code int) {
	defer func() {
		if r := recover(); r != nil {
			multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Could not get exit code, so returning 1 instead (this is non-fatal, but should be resolved), actual error: %v", r)
			code = 128
		}
	}()

	type Status interface {
		ExitStatus() int
	}
	return cmd.ProcessState.Sys().(Status).ExitStatus()
}

// BashifyPath takes a windows %PATH% list and turns it into a bash style PATH list.
// e.g. C:\foo;C:\bar becomes /c/foo:/c/bar
func BashifyPathEnv(pathList string) string {
	// Calling out to bash like in BashifyPath is too slow.
	// Just do a simple string transformation after computing the root prefix.
	// The resulting paths will be valid in bash, they just may not be fully resolved/simplified.
	// This is okay because bash really just needs to find executables to run.
	windir := os.Getenv("WINDIR") // always defined; e.g. "C:\Windows"
	var root string
	if bashified, err := BashifyPath(windir); err == nil { // e.g. "/c/Windows"
		dirNoVol := windir[len(filepath.VolumeName(windir))+1:] // e.g. "Windows"
		if i := strings.Index(bashified, dirNoVol); i > 0 {
			root = bashified[:i-1] // e.g. "/c"
		} else {
			multilog.Error("Could not find windir ('%s') in bashified dir ('%s')", dirNoVol, bashified)
		}
	} else {
		multilog.Error("Could not bashify WINDIR: %v", err)
	}
	if root == "" {
		// Fallback.
		root = "/" + strings.ToLower(filepath.VolumeName(windir)) // e.g. "/c"
	}

	dirs := strings.Split(pathList, ";")
	for i, dir := range dirs {
		dir = dir[len(filepath.VolumeName(dir)):] // "C:\foo bar\baz" -> "\foo bar\baz"
		dirs[i] = root + filepath.ToSlash(dir)    // "C:\foo bar\baz" -> "/c/foo bar/baz"
	}
	return strings.Join(dirs, ":") // bash uses ':' while Windows uses ';'
}

var dynamicEnvVarRe = regexp.MustCompile(`(^=.+)=(.+)`)

// InheritEnv returns a union of the given environment and os.Environ(). If the given environment
// and os.Environ() share any environment variables, the former's will be used over the latter's.
func InheritEnv(env map[string]string) map[string]string {
	for _, kv := range os.Environ() {
		eq := strings.Index(kv, "=")
		key := kv[:eq]
		value := kv[eq+1:]

		// cmd.exe on Windows uses some dynamic environment variables
		// that begin with an '='. We want to make sure we include
		// these in the virtual environment. For more information see:
		// https://devblogs.microsoft.com/oldnewthing/20100506-00/?p=14133
		if strings.HasPrefix(kv, "=") {
			groups := dynamicEnvVarRe.FindStringSubmatch(kv)
			if len(groups) == 0 {
				continue
			}
			env[groups[1]] = groups[2]
		} else {
			// Windows allows environment variables that are not uppercase.
			// This can lead to duplicate path entries. At this point we
			// have already constructed the env vars that we need for
			// our virtual environment so we discard any duplicate entries`.
			if _, ok := env[strings.ToUpper(key)]; ok {
				continue
			}

			if _, ok := env[key]; !ok {
				env[key] = value
			}
		}
	}

	return env
}

// IsAccessDeniedError is primarily used to determine if an operation failed due to insufficient
// permissions (e.g. attempting to kill an admin process as a normal user)
func IsAccessDeniedError(err error) bool {
	return errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
