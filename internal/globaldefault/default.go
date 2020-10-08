package globaldefault

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

const shimDenoter = "(DO NOT EDIT!) State Tool shim for projects using following languages"

const prefPrefix = "default_project_path_"

type DefaultConfigurer interface {
	Set(key string, value interface{})
	GetString(key string) string
}

// rollbackShims removes all shims in the global binary directory that target a previous default project
// If the target of a shim does not exist anymore, the shim is also removed.
func rollbackShims(languages []*language.Language) error {
	binDir := config.GlobalBinPath()

	// remove symlinks pointing to default project
	files, err := ioutil.ReadDir(binDir)
	if err != nil {
		return errs.Wrap(err, "Could not read through global bin dir")
	}
	for _, f := range files {
		fn := filepath.Join(binDir, f.Name())
		shim := newShim(fn)
		if !shim.OneOfLanguage(languages) {
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

// createShimFiles creates shims in the target path of all executables found in the bins dir
// It overwrites existing files, if the overwrite flag is set.
// On Windows the same executable name can have several file extensions,
// therefore executables are only shimmed if it has not been shimmed for a
// target (with the same or a different extension) from a different directory.
// Also: Only the executable with the highest priority according to pathExt is shimmed.
func createShimFiles(exePaths []string, languages []*language.Language) error {
	for _, exePath := range exePaths {
		shim := newShim(exePath)

		if err := shim.Create(languages); err != nil {
			return locale.WrapError(err, "err_createshim", "Could not create shim for {{.V0}} at {{.V1}}.", exePath, shim.Path())
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

	languages, err := runtime.Languages()
	if err != nil {
		return locale.WrapError(err, "err_default_runtime_languages", "Could not figure out what languages belong to the given runtime environment.")
	}

	// roll back old symlinks
	if err := rollbackShims(languages); err != nil {
		return locale.WrapError(err, "err_rollback_shim", "Could not roll back previous shim installation.")
	}

	// Retrieve artifact binary directory
	var bins []string
	if p, ok := envMap["PATH"]; ok {
		bins = strings.Split(p, string(os.PathListSeparator))
	}

	exes, err := exeutils.Executables(bins)
	if err != nil {
		return locale.WrapError(err, "err_symlink_exes", "Could not detect executables")
	}

	// Remove duplicate executables as per PATH and PATHEXT
	exes, err = exeutils.UniqueExes(exes, os.Getenv("PATHEXT"))
	if err != nil {
		return locale.WrapError(err, "err_unique_exes", "Could not detect unique executables, make sure your PATH and PATHEXT environment variables are properly configured.")
	}

	if err := createShimFiles(exes, languages); err != nil {
		return locale.WrapError(err, "err_createshims", "Could not create shim files to set up the default runtime environment.")
	}

	for _, lang := range languages {
		cfg.Set(prefPrefix+lang.String(), runtime.InstallPath())
	}

	return nil
}
