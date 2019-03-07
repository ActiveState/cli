package mock

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/stretchr/testify/mock"
)

// Installer is a testify Mock object.
type Installer struct {
	mock.Mock
}

// NewMockInstaller returns a new testify/mock.Mock Installer.
func NewMockInstaller() *Installer {
	return &Installer{}
}

// InstallDir for Installer.
func (installer *Installer) InstallDir() string {
	args := installer.Called()
	return args.String(0)
}

// Install for Installer.
func (installer *Installer) Install(archivePath string) *failures.Failure {
	args := installer.Called(archivePath)
	return args.Get(0).(*failures.Failure)
}
