package osutils

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

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
	if len(absolutePath) > 0 && absolutePath[0:1] == "/" {
		// Already the format we want
		return absolutePath, nil
	}

	if len(absolutePath) < 2 || absolutePath[1:2] != ":" {
		// Check for windows style paths
		return "", errs.New("Unrecognized absolute path format: %s", absolutePath)
	}

	if strings.Contains(absolutePath, ";") {
		return "", errs.New("Path is a list: %s", absolutePath)
	}

	winPath, err := winPathToLinPath(absolutePath)
	if err == nil {
		winPath = strings.Replace(winPath, ` `, `\ `, -1) // escape space
		return winPath, nil
	}

	// Do not log to rollbar because the path might not exist (yet).
	logging.Error("Failed to bashify path using installed bash executable, falling back to slash replacement: %v", err)

	vol := filepath.VolumeName(absolutePath)
	absolutePath = absolutePath[len(vol):]
	vol = strings.Replace(vol, ":", "", 1)
	winPath = "/" + vol + filepath.ToSlash(absolutePath)
	winPath = strings.Replace(winPath, ` `, `\ `, -1) // escape space
	return winPath, nil
}

func winPathToLinPath(name string) (string, error) {
	cmd := exec.Command("bash", "-c", "pwd")
	cmd.Dir = filepath.Dir(name)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	path := strings.TrimSpace(string(out)) + "/" + filepath.Base(name)

	return path, nil
}

// BashifyPath takes a windows %PATH% list and turns it into a bash style PATH list.
// e.g. C:\foo;C:\bar becomes /c/foo:/c/bar
func BashifyPathEnv(pathList string) (string, error) {
	if runtime.GOOS != "windows" {
		return pathList, nil // already bashified
	}

	dirs := strings.Split(pathList, ";")
	for i, dir := range dirs {
		path, err := BashifyPath(dir)
		if err != nil {
			return "", errs.Wrap(err, "Unable to bashify path: %v", dir)
		}
		dirs[i] = strings.Replace(path, `\ `, " ", -1)
	}
	return strings.Join(dirs, ":"), nil // bash uses ':' while Windows uses ';'
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
func Executable() string {
	exec, err := os.Executable()
	if err != nil {
		exec = os.Args[0]
	}

	resolved, err := fileutils.ResolvePath(exec)
	if err != nil {
		return exec
	}

	return resolved
}

// ExecutableName returns the name of the executable called with the extension
// removed and falls back to the command used to call the executable.
func ExecutableName() string {
	name := filepath.Base(Executable())
	name = strings.TrimSuffix(name, path.Ext(name))
	return name
}
