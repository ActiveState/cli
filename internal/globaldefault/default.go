package globaldefault

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	rt "runtime"
	"strings"

	"github.com/gobuffalo/packr"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type DefaultConfigurer interface {
	Set(key string, value interface{})
	GetString(key string) string
}

type exeFile struct {
	fpath string
	name  string
	ext   string
}

// isShimFor check if the specified filename is (probably) a shim and targets a file in the specified directory
// This function is used during rollback to clean old shims
func isShimFor(filename, dir string) bool {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		logging.Debug("Could not read file contents of shim candidate %s: %v", filename, err)
		return false
	}

	targetRe := regexp.MustCompile("(?m)^(?:REM|#) State Tool Shim Target: (.*)$")
	target := targetRe.FindStringSubmatch(string(contents))

	if len(target) != 2 {
		return false
	}

	if dir == "" {
		return true
	}

	res, err := fileutils.PathContainsParent(target[1], dir)
	if err != nil {
		logging.Debug("Error determining if path %s is child of path %s: %v", target, dir, err)
		return false
	}
	return res
}

// rollbackShims removes all shims in the global binary directory that target a previous default project
// If the target of a shim does not exist anymore, the shim is also removed.
func rollbackShims(cfg DefaultConfigurer) error {
	binDir := config.GlobalBinPath()

	// remove symlinks pointing to default project
	files, err := ioutil.ReadDir(binDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		fn := filepath.Join(binDir, f.Name())
		if !isShimFor(fn, config.CachePath()) {
			continue
		}

		// remove shim if it links to old project path or target does not exist anymore
		err = os.Remove(fn)
		if err != nil {
			return locale.WrapError(err, "rollback_remove_err", "Failed to remove shim {{.V0}}", fn)
		}
	}
	return nil
}

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

// UniqueExes filters the array of executables for those that would be selected by the command shell in case of a name collision
func UniqueExes(exePaths []string, pathext string) ([]string, error) {
	pathExt := strings.Split(strings.ToLower(pathext), ";")
	exeFiles := map[string]exeFile{}
	result := []string{}

	for _, exePath := range exePaths {
		if rt.GOOS == "windows" {
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

// shimTargetPath returns the full path to the shim target (adds .bat on Windows)
func shimTargetPath(targetDir string, path string) string {
	target := filepath.Clean(filepath.Join(targetDir, filepath.Base(path)))
	if rt.GOOS != "windows" {
		return target
	}

	oldExt := filepath.Ext(target)
	return target[0:len(target)-len(oldExt)] + ".bat"
}

func createShimFile(fpath, shimPath string) error {
	logging.Debug("Shimming %s at %s", fpath, shimPath)
	exe, err := os.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not get State Tool executable")
	}

	tplParams := map[string]interface{}{
		"exe":     exe,
		"command": filepath.Base(fpath),
		"target":  fpath,
	}
	box := packr.NewBox("../../assets/shim")
	boxFile := "shim.sh"
	if rt.GOOS == "windows" {
		boxFile = "shim.bat"
	}
	shimBytes := box.Bytes(boxFile)
	shimStr, err := strutils.ParseTemplate(string(shimBytes), tplParams)
	if err != nil {
		return errs.Wrap(err, "Could not parse %s template", boxFile)
	}

	err = ioutil.WriteFile(shimPath, []byte(shimStr), 0755)
	if err != nil {
		return errs.Wrap(err, "failed to write shim command %s", shimPath)
	}
	return nil
}

// createShimFiles creates shims in the target path of all executables found in the bins dir
// It overwrites existing files, if the overwrite flag is set.
// On Windows the same executable name can have several file extensions,
// therefore executables are only shimmed if it has not been shimmed for a
// target (with the same or a different extension) from a different directory.
// Also: Only the executable with the highest priority according to pathExt is shimmed.
func createShimFiles(targetPath string, exePaths []string) error {
	for _, exePath := range exePaths {
		shimPath := shimTargetPath(targetPath, exePath)

		// The link should not exist as we are always rolling back old shims before we run this code.
		if fileutils.TargetExists(shimPath) {
			logging.Error("Cannot create shim as target already exists: %s.", shimPath)
			continue
		}

		if err := createShimFile(exePath, shimPath); err != nil {
			return err
		}
	}

	return nil
}

// SetupDefaultActivation sets symlinks in the global bin directory to the currently activated runtime
func SetupDefaultActivation(cfg DefaultConfigurer, runtime *runtime.Runtime) error {
	env, fail := runtime.Env()
	if fail != nil {
		return errs.Wrap(fail, "Could not get runtime env")
	}

	envMap, err := env.GetEnv(false, "")
	if err != nil {
		return errs.Wrap(err, "Could not get env")
	}

	// roll back old symlinks
	if err := rollbackShims(cfg); err != nil {
		return locale.WrapError(err, "err_rollback_shim", "Could not roll back previous shim installation.")
	}

	// Retrieve artifact binary directory
	var bins []string
	if p, ok := envMap["PATH"]; ok {
		bins = strings.Split(p, string(os.PathListSeparator))
	}

	exes, err := Executables(bins)
	if err != nil {
		return locale.WrapError(err, "err_symlink_exes", "Could not detect executables")
	}

	// Remove duplicate executables as per PATH and PATHEXT
	exes, err = UniqueExes(exes, os.Getenv("PATHEXT"))
	if err != nil {
		return locale.WrapError(err, "err_unique_exes", "Could not detect unique executables, make sure your PATH and PATHEXT environment variables are properly configured.")
	}

	binDir := config.GlobalBinPath()
	return createShimFiles(binDir, exes)
}
