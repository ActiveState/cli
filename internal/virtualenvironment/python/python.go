package python

import (
	"path"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	dataDir   string
	installer runtime.Installer
}

// NewVirtualEnvironment returns a configured python virtualenvironment.
func NewVirtualEnvironment(dataDir string, pythonInstaller runtime.Installer) (*VirtualEnvironment, *failures.Failure) {
	if pythonInstaller == nil {
		return nil, failures.FailInvalidArgument.New("venv_installer_is_nil")
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
