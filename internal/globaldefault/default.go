package globaldefault

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/gobuffalo/packr"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

const shimDenoter = "!DO NOT EDIT! State Tool Shim !DO NOT EDIT!"

type DefaultConfigurer interface {
	sscommon.Configurable
	CachePath() string
}

// BinDir returns the global binary directory
func BinDir(cfg DefaultConfigurer) string {
	return filepath.Join(cfg.CachePath(), "bin")
}

func isBinDirOnWindowsUserPath(binDir string) bool {
	if rt.GOOS != "windows" {
		return false
	}

	cmdEnv := cmd.NewCmdEnv(true)
	path, err := cmdEnv.Get("PATH")
	if err != nil {
		logging.Error("Failed to get user PATH")
		return false
	}

	return funk.ContainsString(
		strings.Split(path, string(os.PathListSeparator)),
		binDir,
	)
}

func Prepare(cfg DefaultConfigurer, subshell subshell.SubShell) error {
	logging.Debug("Preparing globaldefault")
	binDir := BinDir(cfg)

	// Don't run prepare if we're already on PATH
	path := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, p := range path {
		if p == binDir {
			return nil
		}
	}

	isWindowsAdmin, err := osutils.IsWindowsAdmin()
	if err != nil {
		logging.Error("Failed to determine if we are running as administrator: %v", err)
	}
	if isWindowsAdmin {
		logging.Debug("Skip preparation step as it is not supported for Windows Administrators.")
		return nil
	}
	if isBinDirOnWindowsUserPath(binDir) {
		logging.Debug("Skip preparation step as it has been done previously for the current user.")
		return nil
	}

	if err := fileutils.MkdirUnlessExists(binDir); err != nil {
		return locale.WrapError(err, "err_globaldefault_bin_dir", "Could not create bin directory.")
	}

	envUpdates := map[string]string{
		"PATH": binDir,
	}

	if err := subshell.WriteUserEnv(cfg, envUpdates, sscommon.Default, true); err != nil {
		return locale.WrapError(err, "err_globaldefault_update_env", "Could not write to user environment.")
	}

	return nil
}

// WarningForAdministrator prints a warning message if default activation is invoked by a Windows Administrator
// The default activation will only be accessible by the underlying unprivileged user.
func WarningForAdministrator(out output.Outputer) {
	if rt.GOOS != "windows" {
		return
	}

	isAdmin, err := osutils.IsWindowsAdmin()
	if err != nil {
		logging.Error("Failed to determine if run as administrator.")
	}
	if isAdmin {
		u, err := user.Current()
		if err != nil {
			logging.Error("Failed to determine current user.")
			return
		}
		out.Notice(locale.Tl(
			"default_admin_activation_warning",
			"[NOTICE]The default activation is added to the environment of user {{.V0}}.  The project may be inaccessible when run with Administrator privileges or authenticated as a different user.[/RESET]",
			u.Username,
		))
	}
}

// SetupDefaultActivation sets symlinks in the global bin directory to the currently activated runtime
func SetupDefaultActivation(subshell subshell.SubShell, cfg DefaultConfigurer, runtime *runtime.Runtime, projectPath string) error {
	logging.Debug("Setting up globaldefault")
	if err := Prepare(cfg, subshell); err != nil {
		return locale.WrapError(err, "err_globaldefault_prepare", "Could not prepare environment.")
	}

	env, err := runtime.Env()
	if err != nil {
		return errs.Wrap(err, "Could not get runtime env")
	}

	envMap, err := env.GetEnv(false, "")
	if err != nil {
		return errs.Wrap(err, "Could not get env")
	}

	// roll back old symlinks
	if err := cleanup(cfg); err != nil {
		return locale.WrapError(err, "err_rollback_shim", "Could not clean up previous default installation.")
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

	if err := createShims(exes, projectPath, cfg); err != nil {
		return locale.WrapError(err, "err_createshims", "Could not create shim files to set up the default runtime environment.")
	}

	cfg.Set(constants.GlobalDefaultPrefname, projectPath)

	return nil
}

func cleanup(cfg DefaultConfigurer) error {
	binDir := BinDir(cfg)
	if err := fileutils.MkdirUnlessExists(binDir); err != nil {
		return locale.WrapError(err, "err_globaldefault_mkdir", "Could not create bin directory: {{.V0}}.", binDir)
	}

	// remove existing binaries in our bin dir
	files, err := ioutil.ReadDir(binDir)
	if err != nil {
		return errs.Wrap(err, "Could not read through global bin dir")
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fn := filepath.Join(binDir, f.Name())

		err = os.Remove(fn)
		if err != nil {
			return locale.WrapError(err, "rollback_remove_err", "Failed to remove shim {{.V0}}", fn)
		}
	}

	return nil
}

func createShims(exePaths []string, projectPath string, cfg DefaultConfigurer) error {
	for _, exePath := range exePaths {
		if err := createShim(exePath, projectPath, cfg); err != nil {
			return locale.WrapError(err, "err_createshim", "Could not create shim for {{.V0}}.", exePath)
		}
	}

	return nil
}

func createShim(exePath, projectPath string, cfg DefaultConfigurer) error {
	target := filepath.Clean(filepath.Join(BinDir(cfg), filepath.Base(exePath)))
	if rt.GOOS == "windows" {
		oldExt := filepath.Ext(target)
		target = target[0:len(target)-len(oldExt)] + ".bat"
	}
	logging.Debug("Shimming %s at %s", exePath, target)

	// The link should not exist as we are always rolling back old shims before we run this code.
	if fileutils.TargetExists(target) {
		return locale.NewError("err_createshim_exists", "Could not create shim as target already exists: {{.V0}}.", target)
	}

	exe, err := os.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not get State Tool executable")
	}

	tplParams := map[string]interface{}{
		"exe":         exe,
		"command":     filepath.Base(exePath),
		"projectPath": projectPath,
		"denote":      shimDenoter,
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

	if err = ioutil.WriteFile(target, []byte(shimStr), 0755); err != nil {
		return locale.WrapError(err, "Could not create shim for {{.V0}} at {{.V1}}.", exePath, target)
	}

	return nil
}
