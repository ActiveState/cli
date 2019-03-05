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
	if failure := args.Get(1); failure != nil {
		return failure.(*failures.Failure)
	}
	return nil
}
