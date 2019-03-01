package python

import (
	"os"
	"path"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	dataDir  string
	cacheDir string
}

// NewVirtualEnvironment returns a configured python virtualenvironment.
func NewVirtualEnvironment(dataDir string, cacheDir string) *VirtualEnvironment {
	return &VirtualEnvironment{
		dataDir:  dataDir,
		cacheDir: cacheDir,
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

// CacheDir - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) CacheDir() string {
	return v.cacheDir
}

// WorkingDirectory - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) WorkingDirectory() string {
	return ""
}

// Activate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	if err := fileutils.Mkdir(v.dataDir, "bin"); err != nil {
		return err
	}
	return fileutils.Mkdir(v.dataDir, "lib")
}

// Env - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Env() map[string]string {
	return map[string]string{
		"PATH": path.Join(v.cacheDir, "bin") + string(os.PathListSeparator) + path.Join(v.dataDir, "bin"),
	}
}
