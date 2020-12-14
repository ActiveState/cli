package mock

import (
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
func (installer *Installer) Install() error {
	installer.Called()
	return nil
}

// InstallFromArchive for Installer.
func (installer *Installer) InstallFromArchive(archive string) error {
	installer.Called()
	return nil
}

