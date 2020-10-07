package runtime

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gobuffalo/packr"
	"github.com/google/uuid"
	"github.com/vbauerster/mpb/v4"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/model"
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

	// FailNoCommitID represents a Failure due to a missing commit ID
	FailNoCommitID = failures.Type("runtime.runtime.notcommitid", failures.FailUser)

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

	// FailRequiresDownload is a failure due to not all artifacts having been downloaded
	FailRequiresDownload = failures.Type("runtime.requires.download", failures.FailUserInput)

	// FailRuntimeInvalidEnvironment represents a Failure during set up of the runtime environment
	FailRuntimeInvalidEnvironment = failures.Type("runtime.runtime.invalidenv", failures.FailIO)

	// FailRuntimeUnknownEngine is a failure due to the engine not being known.
	FailRuntimeUnknownEngine = failures.Type("runtime.runtime.unknownengine", FailRuntimeInvalid)
)

type MessageHandler interface {
	buildlogstream.MessageHandler
	DownloadStarting()
}

// Installer implements an Installer that works with a runtime.Downloader and a
// runtime.Installer. Effectively, upon calling Install, the Installer will first
// try and Download an archive, then it will try to install that downloaded archive.
type Installer struct {
	runtime           *Runtime
	runtimeDownloader Downloader
}

// NewInstaller creates a new RuntimeInstaller
func NewInstaller(runtime *Runtime) *Installer {
	installer := &Installer{
		runtime:           runtime,
		runtimeDownloader: NewDownload(runtime),
	}

	return installer
}

// Install will download the installer archive and invoke InstallFromArchive
func (installer *Installer) Install() (envGetter EnvGetter, freshInstallation bool, fail *failures.Failure) {
	assembler, fail := installer.Assembler()
	if fail != nil {
		return nil, false, fail
	}
	return installer.InstallArtifacts(assembler)
}

// Env will grab the environment information for the given runtime. This will request build info.
func (installer *Installer) Env() (envGetter EnvGetter, fail *failures.Failure) {
	return installer.Assembler()
}

// IsInstalled will check if the installer has already ran (ie. the artifacts already exist at the target dir)
func (installer *Installer) IsInstalled() (bool, *failures.Failure) {
	assembler, fail := installer.Assembler()
	if fail != nil {
		return false, fail
	}

	return assembler.IsInstalled(), nil
}

// Assembler returns a new runtime assembler for the given checkpoint and artifacts
func (installer *Installer) Assembler() (Assembler, *failures.Failure) {
	if fail := installer.validateCheckpoint(); fail != nil {
		return nil, fail
	}

	artifacts, fail := installer.runtimeDownloader.FetchArtifacts()
	if fail != nil {
		return nil, fail
	}

	switch artifacts.BuildEngine {
	case Alternative:
		return NewAlternativeRuntime(artifacts.Artifacts, installer.runtime.RuntimeDir, artifacts.RecipeID)
	case Camel:
		return NewCamelRuntime(installer.runtime.CommitID, artifacts.Artifacts, installer.runtime.RuntimeDir)
	case Hybrid:
		cr, fail := NewCamelRuntime(installer.runtime.CommitID, artifacts.Artifacts, installer.runtime.RuntimeDir)
		if fail != nil {
			return nil, fail
		}

		return &HybridRuntime{cr}, nil
	default:
		return nil, FailRuntimeUnknownEngine.New("installer_err_engine_unknown")
	}
}

// InstallArtifacts installs all artifacts provided by a runtime assembler
func (installer *Installer) InstallArtifacts(runtimeAssembler Assembler) (envGetter EnvGetter, freshInstallation bool, fail *failures.Failure) {
	if runtimeAssembler.IsInstalled() {
		logging.Debug("runtime already successfully installed")
		return runtimeAssembler, false, nil
	}



	downloadArtfs := runtimeAssembler.ArtifactsToDownload()
	unpackArchives := map[string]*HeadChefArtifact{}
	progressBar := progress.New(mpb.WithOutput(os.Stderr))
	if strings.ToLower(os.Getenv(constants.NonInteractive)) == "true" {
		progressBar = progress.New(mpb.WithOutput(nil))
	}
	defer progressBar.Close()

	if len(downloadArtfs) != 0 {
		if installer.runtime.msgHandler != nil {
			installer.runtime.msgHandler.DownloadStarting()
		}

		if len(downloadArtfs) > 0 {
			archives, fail := installer.runtimeDownloader.Download(downloadArtfs, runtimeAssembler, progressBar)
			if fail != nil {
				progressBar.Cancel()
				return nil, false, fail
			}

			for k, v := range archives {
				unpackArchives[k] = v
			}
		}
	}

	fail = installer.InstallFromArchives(unpackArchives, runtimeAssembler, progressBar)
	if fail != nil {
		progressBar.Cancel()
		return nil, false, fail
	}

	// We still want to run PostInstall because even though no new artifact might be downloaded we still might be
	// deleting some already cached ones
	err := runtimeAssembler.PostInstall()
	if err != nil {
		return nil, false, failures.FailRuntime.Wrap(err, "error during post installation step")
	}

	return runtimeAssembler, true, nil
}

// validateCheckpoint tries to see if the checkpoint has any chance of succeeding
func (installer *Installer) validateCheckpoint() *failures.Failure {
	if installer.runtime.CommitID == "" {
		return FailNoCommitID.New("installer_err_runtime_no_commitid")
	}

	checkpoint, _, fail := model.FetchCheckpointForCommit(installer.runtime.CommitID)
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
func (installer *Installer) InstallFromArchives(archives map[string]*HeadChefArtifact, a Assembler, pg *progress.Progress) *failures.Failure {
	var bar *progress.TotalBar
	if len(archives) > 0 {
		bar = pg.AddTotalBar(locale.T("installing"), len(archives))
	}

	fail := a.PreInstall()
	if fail != nil {
		pg.Cancel()
		return fail
	}

	for archivePath, artf := range archives {
		if fail := installer.InstallFromArchive(archivePath, artf, a, pg); fail != nil {
			pg.Cancel()
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

	installDir := installer.runtime.RuntimeDir
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

// installPPMShim installs an executable shell script and a BAT file that is executed instead of PPM in the specified path.
// It calls the `state _ppm` sub-command printing deprecation messages.
func installPPMShim(binPath string) error {
	// remove old ppm command if it existed before
	ppmExeName := "ppm"
	if runtime.GOOS == "windows" {
		ppmExeName = "ppm.exe"
	}
	ppmExe := filepath.Join(binPath, ppmExeName)
	if fileutils.FileExists(ppmExe) {
		err := os.Remove(ppmExe)
		if err != nil {
			return errs.Wrap(err, "failed to remove existing ppm %s", ppmExe)
		}
	}

	box := packr.NewBox("../../../assets/ppm")
	ppmBytes := box.Bytes("ppm.sh")
	shim := filepath.Join(binPath, "ppm")
	// remove shim if it existed before, so we can overwrite (ok to drop error here)
	_ = os.Remove(shim)

	exe, err := os.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not get executable")
	}

	tplParams := map[string]interface{}{"exe": exe}
	ppmStr, err := strutils.ParseTemplate(string(ppmBytes), tplParams)
	if err != nil {
		return errs.Wrap(err, "Could not parse ppm.sh template")
	}

	err = ioutil.WriteFile(shim, []byte(ppmStr), 0755)
	if err != nil {
		return errs.Wrap(err, "failed to write shim command %s", shim)
	}
	if runtime.GOOS == "windows" {
		ppmBatBytes := box.Bytes("ppm.bat")
		shim := filepath.Join(binPath, "ppm.bat")
		// remove shim if it existed before, so we can overwrite (ok to drop error here)
		_ = os.Remove(shim)

		ppmBatStr, err := strutils.ParseTemplate(string(ppmBatBytes), tplParams)
		if err != nil {
			return errs.Wrap(err, "Could not parse ppm.sh template")
		}

		err = ioutil.WriteFile(shim, []byte(ppmBatStr), 0755)
		if err != nil {
			return errs.Wrap(err, "failed to write shim command %s", shim)
		}
	}

	return nil
}
