package python

import (
	"path"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/installer"
)

// NewInstaller creates a new installer.RuntimeInstaller which can install python for this virtualenvironment.
func NewInstaller(targetDir string) (installer.Installer, *failures.Failure) {
	apyInstaller, failure := runtime.NewActivePythonInstaller(targetDir)
	if failure != nil {
		return nil, failure
	}
	return installer.NewRuntimeInstaller(runtime.InitRuntimeDownload(targetDir), apyInstaller), nil
}

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	dataDir   string
	installer installer.Installer
}

// NewVirtualEnvironment returns a configured python virtualenvironment.
func NewVirtualEnvironment(dataDir string, pythonInstaller installer.Installer) (*VirtualEnvironment, *failures.Failure) {
	if pythonInstaller == nil {
		return nil, failures.FailVerify.New("venv_installer_is_nil")
	}

	return &VirtualEnvironment{
		dataDir:   dataDir,
		installer: pythonInstaller,
	}, nil
}

// Language - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Language() string {
	return "python3"
}

// DataDir - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) DataDir() string {
	return v.dataDir
}

// WorkingDirectory - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) WorkingDirectory() string {
	return ""
}

// Activate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	if isEmpty, failure := fileutils.IsEmptyDir(v.dataDir); failure != nil {
		return failure
	} else if isEmpty {
		return v.installer.Install()
	}

	return nil
}

// Env - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Env() map[string]string {
	return map[string]string{
		"PATH": path.Join(v.dataDir, "bin"),
	}
}
