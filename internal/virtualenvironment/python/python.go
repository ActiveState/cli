package python

import (
	"path"

	"github.com/ActiveState/cli/internal/failures"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	dataDir string
}

// NewVirtualEnvironment returns a configured python virtualenvironment.
func NewVirtualEnvironment(dataDir string) *VirtualEnvironment {
	return &VirtualEnvironment{
		dataDir: dataDir,
	}
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
	return nil
}

// Env - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Env() map[string]string {
	return map[string]string{
		"PATH": path.Join(v.dataDir, "bin"),
	}
}
