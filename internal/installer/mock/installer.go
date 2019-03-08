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

// Install for Installer.
func (installer *Installer) Install() *failures.Failure {
	args := installer.Called()
	if failure := args.Get(0); failure != nil {
		return failure.(*failures.Failure)
	}
	return nil
}

// OnDownload registers a function to be called when a download occurs
func (installer *Installer) OnDownload(f func()) {}

// OnInstall registers a function to be called when an install occurs
func (installer *Installer) OnInstall(f func()) {}
