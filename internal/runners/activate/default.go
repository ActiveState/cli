package activate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/gobuffalo/packr"
	"github.com/thoas/go-funk"
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

func getDefaultProjectPath(d DefaultConfigurer) string {
	return d.GetString("default_project_path")
}

func setDefaultProjectPath(d DefaultConfigurer, path string) {
	d.Set("default_project_path", path)
}

type primable interface {
	primer.Outputer
}

// gets the intended target for a shim
func shimTarget(fn string) (string, error) {
	contents, err := ioutil.ReadFile(fn)
	if err != nil {
		return "", err
	}

	targetRe := regexp.MustCompile("^ target: (.)$")
	target := targetRe.FindString(string(contents))
	if target == "" {
		return "", errs.New("Target file is not a shim.")
	}

	return target, nil
}

// rollbackShims removes all shims in the global binary directory that target a previous default project
// If the target of a shim does not exist anymore, the shim is also removed.
func rollbackShims(cfg DefaultConfigurer) error {
	projectPath := getDefaultProjectPath(cfg)
	binDir := config.GlobalBinPath()

	// remove symlinks pointing to default project
	files, err := ioutil.ReadDir(binDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		fn := f.Name()
		target, err := shimTarget(fn)
		if err != nil {
			continue
		}

		// TODO: Decide if we need to do these checks...
		if fileutils.TargetExists(target) && (projectPath == "" || !strings.HasPrefix(target, projectPath)) {
			continue
		}

		// remove shim if it links to old project path or target does not exist anymore
		err = os.Remove(fn)
		if err != nil {
			return locale.WrapError(err, "rollback_remove_err", "Failed to remove symlink {{.V0}}", fn)
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

func shim(fpath, shimPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not get State Tool executable")
	}

	tplParams := map[string]interface{}{
		"exe":     exe,
		"command": filepath.Base(fpath),
		"target":  fpath,
	}
	box := packr.NewBox("../../../assets/shim")
	var shimStr string
	if rt.GOOS != "windows" {
		shimBytes := box.Bytes("shim.sh")
		shimStr, err = strutils.ParseTemplate(string(shimBytes), tplParams)
		if err != nil {
			return errs.Wrap(err, "Could not parse shim.sh template")
		}
	} else {
		shimBytes := box.Bytes("shim.bat")
		shimStr, err = strutils.ParseTemplate(string(shimBytes), tplParams)
		if err != nil {
			return errs.Wrap(err, "Could not parse shim.bat template")
		}
	}

	err = ioutil.WriteFile(shimPath, []byte(shimStr), 0755)
	if err != nil {
		return errs.Wrap(err, "failed to write shim command %s", shimPath)
	}
	return nil
}

func needsRollback() bool {
	return true
}

// shimsWithTarget creates shims in the target path of all executables found in the bins dir
// It overwrites existing files, if the overwrite flag is set.
// On Windows the same executable name can have several file extensions,
// therefore executables are only shimmed if it has not been shimmed for a
// target (with the same or a different extension) from a different directory.
// Also: Only the executable with the highest priority according to pathExt is shimmed.
func shimsWithTarget(targetPath string, exePaths []string, out output.Outputer) error {
	out.Print(locale.Tl("default_shim", "Writing default installation to {{.V0}}.", targetPath))
	for _, exePath := range exePaths {
		shimPath := shimTargetPath(targetPath, exePath)

		// The link should not exist as we are always rolling back old shims before we run this code.
		if fileutils.TargetExists(shimPath) {
			return locale.NewInputError(
				"err_default_symlink_target_exists",
				"Cannot create shim as the target already exists: {{.V0}}.", shimPath)
		}

		logging.Debug("Shimming %s at %s", exePath, shimPath)
		if err := shim(exePath, shimPath); err != nil {
			return err
		}
	}

	return nil
}

// SetupDefaultActivation sets symlinks in the global bin directory to the currently activated runtime
func SetupDefaultActivation(cfg DefaultConfigurer, output output.Outputer, envGetter runtime.EnvGetter) error {
	env, err := envGetter.GetEnv(false, "")
	if err != nil {
		return err
	}

	if needsRollback() {
		// roll back old symlinks
		rollbackShims(cfg)
	}

	// Retrieve artifact binary directory
	var bins []string
	if p, ok := env["PATH"]; ok {
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

	return shimsWithTarget(binDir, exes, output)
}
