package installer

import (
	"path"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

// Installer defines the behavior a generic installer must implement.
type Installer interface {
	// Install will perform an installation.
	Install() *failures.Failure
	OnDownload(func())
	OnInstall(func())
}

// RuntimeInstaller implements an Installer that works with a runtime.Downloader and a
// runtime.Installer. Effectively, upon calling Install, the RuntimeInstaller will first
// try and Download an archive, then it will try to install that downloaded archive.
type RuntimeInstaller struct {
	runtimeDownloader runtime.Downloader
	runtimeInstaller  runtime.Installer
	onDownload        func()
	onInstall         func()
}

// NewRuntimeInstaller creates a new RuntimeInstaller given the provided runtime.Downloader
// and runtime.Installer.
func NewRuntimeInstaller(downloader runtime.Downloader, installer runtime.Installer) *RuntimeInstaller {
	return &RuntimeInstaller{
		runtimeDownloader: downloader,
		runtimeInstaller:  installer,
	}
}

// Install will try to Download an archive using the given runtime.Downloader, install that
// downloaded archive using the given runtime.Installer, or return a Failure if either of
// those actions fail.
func (installer *RuntimeInstaller) Install() *failures.Failure {
	if installer.onDownload != nil {
		installer.onDownload()
	}
	archivePath, failure := installer.runtimeDownloader.Download()
	if failure != nil {
		return failure
	}

	if installer.onInstall != nil {
		installer.onInstall()
	}
	return installer.runtimeInstaller.Install(path.Join(installer.runtimeInstaller.InstallDir(), archivePath))
}

// OnDownload registers a function to be called when a download occurs
func (installer *RuntimeInstaller) OnDownload(f func()) { installer.onDownload = f }

// OnInstall registers a function to be called when an install occurs
func (installer *RuntimeInstaller) OnInstall(f func()) { installer.onInstall = f }
