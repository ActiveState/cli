package runtime

import (
	"os"
	"path/filepath"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// During installation after all files are unpacked to a temporary directory, the progress bar should advanced this much.
// This leaves room to advance the progress bar further while renaming strings in the unpacked files.
const percentReportedAfterUnpack = 85

var (
	// FailInstallDirInvalid represents a Failure due to the working-dir for an installation being invalid in some way.
	FailInstallDirInvalid = failures.Type("runtime.installdir.invalid", failures.FailIO)

	// FailArchiveInvalid represents a Failure due to the installer archive file being invalid in some way.
	FailArchiveInvalid = failures.Type("runtime.archive.invalid", failures.FailIO)

	// FailArchiveNoInstallDir represents a Failure due to an archive not having an install dir
	FailArchiveNoInstallDir = failures.Type("runtime.archive.noinstalldir", FailArchiveInvalid)

	// FailRuntimeInvalid represents a Failure due to a runtime being invalid in some way prior to installation.
	FailRuntimeInvalid = failures.Type("runtime.runtime.invalid", failures.FailIO)

	// FailRuntimeInvalidEnvironment represents a Failure during set up of the runtime environment
	FailRuntimeInvalidEnvironment = failures.Type("runtime.runtime.invalidenv", failures.FailIO)

	// FailNoCommits represents a Failure due to a project not having commits yet (and thus no runtime).
	FailNoCommits = failures.Type("runtime.runtime.nocommits", failures.FailUser)

	// FailPrePlatformNotSupported represents a Failure due to the runtime containing pre-platform bits.
	FailPrePlatformNotSupported = failures.Type("runtime.runtime.preplatform", failures.FailUser)

	// FailRuntimeInstallation represents a Failure to install a runtime.
	FailRuntimeInstallation = failures.Type("runtime.runtime.installation", failures.FailOS)

	// FailRuntimeNotExecutable represents a Failure due to a required file not being executable
	FailRuntimeNotExecutable = failures.Type("runtime.runtime.notexecutable", FailRuntimeInvalid)

	// FailRuntimeNoExecutable represents a Failure due to there not being an executable
	FailRuntimeNoExecutable = failures.Type("runtime.runtime.noexecutable", FailRuntimeInvalid)

	// FailRuntimeNoPrefixes represents a Failure due to there not being any prefixes for relocation
	FailRuntimeNoPrefixes = failures.Type("runtime.runtime.noprefixes", FailRuntimeInvalid)
)

// Installer implements an Installer that works with a runtime.Downloader and a
// runtime.Installer. Effectively, upon calling Install, the Installer will first
// try and Download an archive, then it will try to install that downloaded archive.
type Installer struct {
	cacheDir          string
	runtimeDownloader Downloader
	onDownload        func()
	onInstall         func()
}

// InitInstaller creates a new RuntimeInstaller
func InitInstaller() (*Installer, *failures.Failure) {
	logging.Debug("cache path: %s", config.CachePath())
	return NewInstaller(config.CachePath(), InitDownload())
}

// NewInstaller creates a new RuntimeInstaller after verifying the provided install-dir
// exists as a directory or can be created.
func NewInstaller(cacheDir string, downloader Downloader) (*Installer, *failures.Failure) {
	installer := &Installer{
		cacheDir:          cacheDir,
		runtimeDownloader: downloader,
	}

	return installer, nil
}

// Install will download the installer archive and invoke InstallFromArchive
func (installer *Installer) Install() (envGetter EnvGetter, freshInstallation bool, fail *failures.Failure) {
	if fail := installer.validateCheckpoint(); fail != nil {
		return nil, false, fail
	}

	artifacts, fail := installer.runtimeDownloader.FetchArtifacts()
	if fail != nil {
		return nil, false, fail
	}

	return installer.InstallArtifacts(artifacts)
}

func (installer *Installer) InstallArtifacts(artifactsResult *FetchArtifactsResult) (envGetter EnvGetter, freshInstallation bool, fail *failures.Failure) {
	var runtimeAssembler Assembler
	if artifactsResult.IsAlternative {
		runtimeAssembler, fail = NewAlternativeRuntime(artifactsResult.Artifacts, installer.cacheDir, artifactsResult.RecipeID)
	} else {
		runtimeAssembler, fail = NewCamelRuntime(artifactsResult.Artifacts, installer.cacheDir)
	}
	if fail != nil {
		return nil, false, fail
	}

	downloadArtfs, unpackArchives := runtimeAssembler.ArtifactsToDownloadAndUnpack()

	if len(downloadArtfs) == 0 && len(unpackArchives) == 0 {
		// Already installed, no need to download or install
		logging.Debug("Nothing to download")
		return runtimeAssembler, false, nil
	}

	if installer.onDownload != nil {
		installer.onDownload()
	}

	progress := progress.New()
	defer progress.Close()

	if len(downloadArtfs) > 0 {
		archives, fail := installer.runtimeDownloader.Download(downloadArtfs, runtimeAssembler, progress)
		if fail != nil {
			progress.Cancel()
			return nil, false, fail
		}

		for k, v := range archives {
			unpackArchives[k] = v
		}
	}

	fail = installer.InstallFromArchives(unpackArchives, runtimeAssembler, progress)
	if fail != nil {
		progress.Cancel()
		return nil, false, fail
	}

	return runtimeAssembler, true, nil
}

// validateCheckpoint tries to see if the checkpoint has any chance of succeeding
func (installer *Installer) validateCheckpoint() *failures.Failure {
	pj := project.Get()
	if pj.CommitID() == "" {
		return FailNoCommits.New("installer_err_runtime_no_commits", model.ProjectURL(pj.Owner(), pj.Name(), ""))
	}

	checkpoint, _, fail := model.FetchCheckpointForCommit(strfmt.UUID(pj.CommitID()))
	if fail != nil {
		return fail
	}

	for _, change := range checkpoint {
		if model.NamespaceMatch(change.Namespace, model.NamespacePrePlatformMatch) {
			return FailPrePlatformNotSupported.New("installer_err_runtime_preplatform")
		}
	}

	return nil
}

// InstallFromArchives will unpack the installer archive, locate the install script, and then use the installer
// script to install a runtime to the configured runtime dir. Any failures during this process will result in a
// failed installation and the install-dir being removed.
func (installer *Installer) InstallFromArchives(archives map[string]*HeadChefArtifact, a Assembler, progress *progress.Progress) *failures.Failure {
	bar := progress.AddTotalBar(locale.T("installing"), len(archives))

	fail := a.PreInstall()
	if fail != nil {
		progress.Cancel()
		return fail
	}

	for archivePath, artf := range archives {
		if fail := installer.InstallFromArchive(archivePath, artf, a, progress); fail != nil {
			progress.Cancel()
			return fail
		}
		bar.Increment()
	}

	return nil
}

// InstallFromArchive will unpack artifact and install it
func (installer *Installer) InstallFromArchive(archivePath string, artf *HeadChefArtifact, a Assembler, progress *progress.Progress) *failures.Failure {

	fail := a.PreUnpackArtifact(artf)
	if fail != nil {
		return fail
	}

	installDir := a.InstallationDirectory(artf)
	tmpRuntimeDir, upb, fail := installer.unpackArchive(a.Unarchiver(), archivePath, installDir, progress)
	if fail != nil {
		removeInstallDir(installDir)
		return fail
	}

	fail = a.PostUnpackArtifact(artf, tmpRuntimeDir, archivePath, func() { upb.Increment() })
	if fail != nil {
		removeInstallDir(installDir)
		return fail
	}
	upb.Complete()

	return nil
}

func (installer *Installer) unpackArchive(ua unarchiver.Unarchiver, archivePath string, installDir string, p *progress.Progress) (string, *progress.UnpackBar, *failures.Failure) {
	if fail := installer.validateArchive(ua, archivePath); fail != nil {
		return "", nil, fail
	}

	tmpRuntimeDir := filepath.Join(installDir, uuid.New().String())

	logging.Debug("Unarchiving %s", archivePath)

	// During unpacking we count the number of files to unpack
	var numUnpackedFiles int
	ua.SetNotifier(func(_ string, _ int64, isDir bool) {
		if !isDir {
			numUnpackedFiles++
		}
	})

	// Prepare destination directory and open the archive file
	archiveFile, archiveSize, err := ua.PrepareUnpacking(archivePath, tmpRuntimeDir)
	logging.Debug("Unarchiving %s -> %s %d\n\n\n", archivePath, tmpRuntimeDir, archiveSize)
	if err != nil {
		return tmpRuntimeDir, nil, FailArchiveInvalid.Wrap(err)

	}
	defer archiveFile.Close()

	// create an unpack bar and wrap the archiveFile, when we are done unpacking the
	// bar should say `percentReportedAfterUnpack`%.
	upb := p.AddUnpackBar(archiveSize, percentReportedAfterUnpack)
	wrappedStream := progress.NewReaderProxy(upb.Bar(), upb, archiveFile)

	// unpack it
	logging.Debug("Unarchiving to: %s", tmpRuntimeDir)
	err = ua.Unarchive(wrappedStream, archiveSize, tmpRuntimeDir)
	if err != nil {
		return tmpRuntimeDir, nil, FailArchiveInvalid.Wrap(err)
	}

	// report that we are unpacked.
	upb.Complete()

	logging.Debug("Unpacked %d files\n", numUnpackedFiles)

	// We rescale the progress bar, such that after all files are touched once,
	// we reach 100%  (touching here means, renaming strings in Relocate())
	upb.ReScale(numUnpackedFiles)

	return tmpRuntimeDir, upb, nil
}

// OnDownload registers a function to be called when a download occurs
func (installer *Installer) OnDownload(f func()) {
	installer.onDownload = f
}

// validateArchive ensures the given path to archive is an actual file and that its suffix is a well-known
// suffix for tar+gz files.
func (installer *Installer) validateArchive(ua unarchiver.Unarchiver, archivePath string) *failures.Failure {
	if !fileutils.FileExists(archivePath) {
		return FailArchiveInvalid.New("installer_err_archive_notfound", archivePath)
	} else if err := ua.CheckExt(archivePath); err != nil {
		return FailArchiveInvalid.New("installer_err_archive_badext", archivePath)
	}
	return nil
}

func removeInstallDir(installDir string) {
	if err := os.RemoveAll(installDir); err != nil {
		logging.Errorf("attempting to remove install dir '%s': %v", installDir, err)
	}
}
