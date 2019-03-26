package mock

import (
	"github.com/ActiveState/cli/internal/failures"
	testifyMock "github.com/stretchr/testify/mock"
)

// Installer is a testify Mock object.
type Installer struct {
	testifyMock.Mock
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
func (installer *Installer) Install() *failures.Failure {
	installer.Called()
	return nil
}

// InstallFromArchive for Installer.
func (installer *Installer) InstallFromArchive(archive string) *failures.Failure {
	installer.Called()
	return nil
}

// OnDownload registers a function to be called when a download occurs
func (installer *Installer) OnDownload(f func()) {}

// OnInstall registers a function to be called when an install occurs
func (installer *Installer) OnInstall(f func()) {}
