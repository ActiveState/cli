package activate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/runtime"
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

// TODO sync with Mike's PR
func getGlobalBinDir(d DefaultConfigurer) (string, error) {
	binDir := d.GetString("global_bin_dir")
	if !fileutils.DirExists(binDir) {
		return binDir, errs.New("Could not find default installation directory.")
	}
	return binDir, nil
}

func getDefaultProjectPath(d DefaultConfigurer) string {
	return d.GetString("default")
}

func setDefaultProjectPath(d DefaultConfigurer, path string) {
	d.Set("default", path)
}

type primable interface {
	primer.Outputer
}

// NewDefaultActivation initializes a DefaultActivation struct
// TODO: implement for windows
func isLink(fn string) bool {
	return fileutils.IsSymlink(fn)
}

// TODO: implement for windows
func linkTarget(fn string) (string, error) {
	return fileutils.ResolvePath(fn)
}

// TODO: implement for windows
func link(fpath, symlink string) error {
	err := os.Symlink(fpath, symlink)
	if err != nil {
		return locale.WrapInputError(
			err, "err_default_symlink",
			"Cannot create symlink at {{.V0}}, ensure you have permissions to write to {{.V1}}.", symlink, filepath.Dir(symlink),
		)
	}
	return nil
}

func rollbackSymlinks(config DefaultConfigurer) error {
	projectPath := getDefaultProjectPath(config)
	binDir, err := getGlobalBinDir(config)
	if err != nil {
		return err
	}

	// remove symlinks pointing to default project
	files, err := ioutil.ReadDir(binDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		fn := f.Name()
		if !isLink(fn) {
			continue
		}
		target, err := linkTarget(fn)
		if err != nil {
			return locale.WrapError(err, "rollback_resolve_err", "Failed to resolve target of link {{.V0}}", fn)
		}
		if fileutils.TargetExists(target) && (projectPath == "" || !strings.HasPrefix(target, projectPath)) {
			continue
		}

		// remove symlink if it links to old project path or target does not exist anymore
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

// SymlinkName adds the .lnk file ending on windows
func SymlinkName(targetDir string, path string) string {
	target := filepath.Clean(filepath.Join(targetDir, filepath.Base(path)))
	if rt.GOOS != "windows" {
		return target
	}

	oldExt := filepath.Ext(target)
	return target[0:len(target)-len(oldExt)] + ".lnk"
}

func needsRollback() bool {
	return true
}

// symlinkWithTarget creates symlinks in the target path of all executables found in the bins dir
// It overwrites existing files, if the overwrite flag is set.
// On Windows the same executable name can have several file extensions,
// therefore executables are only symlinked if it has not been symlinked to a
// target (with the same or a different extension) from a different directory.
// Also: Only the executable with the highest priority according to pathExt is symlinked.
func symlinkWithTarget(symlinkPath string, exePaths []string, out output.Outputer) error {
	for _, exePath := range exePaths {
		symlink := SymlinkName(symlinkPath, exePath)

		// If the link already exists we may have to overwrite it, skip it, or fail..
		if fileutils.TargetExists(symlink) {
			// This should not happen as we are always deleting old symlinks before
			return locale.NewInputError(
				"err_default_symlink_target_exists",
				"Cannot create symlink as the target already exists: {{.V0}}.", symlink)
		}

		if err := link(exePath, symlink); err != nil {
			return err
		}
	}

	return nil
}

// SetupDefaultActivation sets symlinks in the global bin directory to the currently activated runtime
func SetupDefaultActivation(config DefaultConfigurer, output output.Outputer, envGetter runtime.EnvGetter) error {
	env, err := envGetter.GetEnv(false, "")
	if err != nil {
		return err
	}

	if needsRollback() {
		// roll back old symlinks
		rollbackSymlinks(config)
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

	binDir, err := getGlobalBinDir(config)
	if err != nil {
		return err
	}

	return symlinkWithTarget(binDir, exes, output)
}
