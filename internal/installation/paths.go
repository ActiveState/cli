package installation

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
)

const (
	// CfgInstallPath is the configuration key for the path where the State Tool is installed
	CfgInstallPath = "installation_path"

	// CfgTransitionalStateToolPath is the configuration key for the path where a transitional State Tool might still be stored
	CfgTransitionalStateToolPath = "transitional_installation_path"

	BinDirName = "bin"

	InstallDirMarker = ".install_root"
)

var ErrCorruptedInstall = errs.New("Corrupted install")

func DefaultInstallPath() (string, error) {
	return InstallPathForBranch(constants.BranchName)
}

func InstallPath() (string, error) {
	// Facilitate use-case of running executables from the build dir while developing
	if !condition.BuiltViaCI() && strings.Contains(os.Args[0], "/build/") {
		return filepath.Dir(os.Args[0]), nil
	}
	if path, ok := os.LookupEnv(constants.OverwriteDefaultInstallationPathEnvVarName); ok {
		return path, nil
	}

	// If State Tool is already exists then we should detect the install path from there
	stateInfo := appinfo.StateApp()
	activeStateOwnedPath := strings.Contains(strings.ToLower(stateInfo.Exec()), "activestate")
	if fileutils.TargetExists(stateInfo.Exec()) {
		if filepath.Base(filepath.Dir(stateInfo.Exec())) == BinDirName && activeStateOwnedPath {
			return filepath.Dir(filepath.Dir(stateInfo.Exec())), nil // <return this>/bin/state.exe
		}
		return filepath.Dir(stateInfo.Exec()), nil // <return this>/state.exe
	}

	return DefaultInstallPath()
}

func BinPathFromInstallPath(installPath string) (string, error) {
	if installPath == "" {
		var err error
		installPath, err = InstallPath()
		if err != nil {
			return installPath, errs.Wrap(err, "Could not detect InstallPath while searching for BinPath")
		}
	}

	return filepath.Join(installPath, BinDirName), nil
}

func InstalledOnPath(installPath string) (bool, string, error) {
	binPath, err := BinPathFromInstallPath(installPath)
	if err != nil {
		return false, "", errs.Wrap(err, "Could not detect binPath from BinPathFromInstallPath")
	}
	fmt.Println("Bin path:", binPath)
	path := appinfo.StateApp(binPath).Exec()
	fmt.Println("Exec path:", path)
	fmt.Println("Exists:", fileutils.TargetExists(path))
	return fileutils.TargetExists(path), path, nil
}

func LauncherInstallPath() (string, error) {
	if path, ok := os.LookupEnv(constants.OverwriteDefaultSystemPathEnvVarName); ok {
		return path, nil
	}
	return defaultSystemInstallPath()
}

func IsInstallRoot(dir string) bool {
	return fileutils.FileExists(filepath.Join(dir, InstallDirMarker))
}

// DetectCorruptedInstallDir will return an error if it detects that the given install path is not a proper
// State Tool installation path. This mainly covers cases where we are working off of a legacy install of the State
// Tool or cases where the uninstall was not completed properly.
func DetectCorruptedInstallDir(path string) error {
	if !fileutils.TargetExists(path) {
		return nil
	}

	isEmpty, err := fileutils.IsEmptyDir(path)
	if err != nil {
		return errs.Wrap(err, "Could not check if install dir is empty")
	}
	if isEmpty {
		return nil
	}

	// Detect if bin dir exists
	binPath, err := BinPathFromInstallPath(path)
	if err != nil {
		return errs.Wrap(err, "Could not detect bin path")
	}
	if !fileutils.DirExists(binPath) {
		return errs.Wrap(ErrCorruptedInstall, "Bin path does not exist: %s", binPath)
	}

	// Detect if the install dir has files in it
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return errs.Wrap(err, "Could not read directory: %s", path)
	}

	for _, file := range files {
		if !file.IsDir() || strings.ToLower(file.Name()) != InstallDirMarker {
			return errs.Wrap(ErrCorruptedInstall, "Install directory should only contain dirs: %s", path)
		}
	}

	// Ensure that bin dir has at least the state and state-svc executables
	files, err = ioutil.ReadDir(binPath)
	if err != nil {
		return errs.Wrap(err, "Could not read bin directory: %s", path)
	}

	var found int
	for _, file := range files {
		fname := strings.ToLower(file.Name())
		if fname == constants.StateCmd+exeutils.Extension || fname == constants.StateSvcCmd+exeutils.Extension {
			found++
		}
	}

	if found != 2 {
		return errs.Wrap(ErrCorruptedInstall, "Bin path did not contain state tool executables.")
	}

	return nil
}
