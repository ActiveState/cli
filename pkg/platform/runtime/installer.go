package runtime

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/gobuffalo/packr"
	"github.com/google/uuid"
	"github.com/vbauerster/mpb/v4"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// During installation after all files are unpacked to a temporary directory, the progress bar should advanced this much.
// This leaves room to advance the progress bar further while renaming strings in the unpacked files.
const percentReportedAfterUnpack = 85

type ErrInstallDirInvalid struct{ *locale.LocalizedError }

type ErrArchiveInvalid struct{ *locale.LocalizedError }

type ErrPrePlatform struct{ *locale.LocalizedError }

type ErrNotExecutable struct{ *locale.LocalizedError }

type ErrNoPrefixes struct{ *locale.LocalizedError }

type MessageHandler interface {
	buildlogstream.MessageHandler
	ChangeSummary(map[strfmt.UUID][]strfmt.UUID, map[strfmt.UUID][]strfmt.UUID, map[strfmt.UUID]*inventory_models.ResolvedIngredient)
	DownloadStarting()
	InstallStarting()
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
func (installer *Installer) Install() (envGetter EnvGetter, freshInstallation bool, err error) {
	if installer.runtime.IsCachedRuntime() {
		ar, err := installer.RuntimeEnv()
		if err == nil {
			return ar, true, nil
		}
		logging.Error("Failed to retrieve cached assembler: %v", err)
	}
	assembler, err := installer.Assembler()
	if err != nil {
		return nil, false, err
	}
	return installer.InstallArtifacts(assembler)
}

// Env will grab the environment information for the given runtime. This will request build info.
func (installer *Installer) Env() (envGetter EnvGetter, err error) {
	if installer.runtime.IsCachedRuntime() {
		ar, err := installer.RuntimeEnv()
		if err == nil {
			return ar, nil
		}
		logging.Error("Failed to retrieve cached assembler: %v", err)
	}
	return installer.Assembler()
}

// IsInstalled will check if the installer has already ran (ie. the artifacts already exist at the target dir)
func (installer *Installer) IsInstalled() (bool, error) {
	if installer.runtime.IsCachedRuntime() {
		return true, nil
	}
	assembler, err := installer.Assembler()
	if err != nil {
		return false, err
	}

	return assembler.IsInstalled(), nil
}

// RuntimeEnv returns the runtime environment specialization all constructed from cached values
func (installer *Installer) RuntimeEnv() (EnvGetter, error) {
	buildEngine, err := installer.runtime.BuildEngine()
	if err != nil {
		return nil, locale.WrapError(err, "installer_err_engine_unknown")
	}
	switch buildEngine {
	case Alternative:
		return NewAlternativeEnv(installer.runtime.runtimeDir)
	case Camel:
		return NewCamelEnv(installer.runtime.commitID, installer.runtime.runtimeDir)
	case Hybrid:
		cr, err := NewCamelEnv(installer.runtime.commitID, installer.runtime.runtimeDir)
		if err != nil {
			return nil, err
		}

		return &HybridRuntime{cr}, nil
	default:
		return nil, locale.NewError("installer_err_engine_unknown")
	}
}

// Assembler returns a new runtime assembler for the given checkpoint and artifacts
func (installer *Installer) Assembler() (Assembler, error) {
	if err := installer.validateCheckpoint(); err != nil {
		return nil, err
	}

	var project *mono_models.Project
	if installer.runtime.Owner() != "" && installer.runtime.ProjectName() != "" {
		var err error
		project, err = model.FetchProjectByName(installer.runtime.Owner(), installer.runtime.ProjectName())
		if err != nil {
			return nil, errs.Wrap(err, "Fetch project failed")
		}
	}

	recipe, err := model.ResolveRecipe(installer.runtime.commitID, installer.runtime.owner, installer.runtime.projectName, project)
	if err != nil {
		return nil, errs.Wrap(err, "ResolveRecipe failed")
	}

	// Run Change Summary
	ingredientMap := model.IngredientVersionMap(recipe)
	directDeps, recursiveDeps := model.ParseDepTree(recipe.ResolvedIngredients, ingredientMap)
	installer.runtime.msgHandler.ChangeSummary(directDeps, recursiveDeps, ingredientMap)

	artifacts, err := installer.runtimeDownloader.FetchArtifacts(recipe, project)
	if err != nil {
		return nil, err
	}

	switch artifacts.BuildEngine {
	case Alternative:
		return NewAlternativeInstall(installer.runtime.runtimeDir, artifacts.Artifacts, artifacts.RecipeID)
	case Camel:
		return NewCamelInstall(installer.runtime.commitID, installer.runtime.runtimeDir, artifacts.Artifacts)
	case Hybrid:
		ci, err := NewCamelInstall(installer.runtime.commitID, installer.runtime.runtimeDir, artifacts.Artifacts)
		if err != nil {
			return nil, err
		}

		return &HybridInstall{ci}, nil
	default:
		return nil, locale.NewError("installer_err_engine_unknown")
	}
}

// InstallArtifacts installs all artifacts provided by a runtime assembler
func (installer *Installer) InstallArtifacts(runtimeAssembler Assembler) (envGetter EnvGetter, freshInstallation bool, err error) {
	if runtimeAssembler.IsInstalled() {
		// write complete marker and build engine files in case they don't exist yet
		err := installer.runtime.MarkInstallationComplete()
		if err != nil {
			return nil, false, locale.WrapError(err, "installer_mark_complete_err", "Failed to mark the installation as complete.")
		}
		err = installer.runtime.StoreBuildEngine(runtimeAssembler.BuildEngine())
		if err != nil {
			return nil, false, locale.WrapError(err, "installer_store_build_engine_err", "Failed to store build engine value.")
		}

		logging.Debug("runtime already successfully installed")
		return runtimeAssembler, false, nil
	}

	downloadArtfs := runtimeAssembler.ArtifactsToDownload()
	unpackArchives := map[string]*HeadChefArtifact{}

	progressOut := os.Stderr
	if strings.ToLower(os.Getenv(constants.NonInteractive)) == "true" {
		progressOut = nil
	}

	downloadProgress := progress.New(mpb.WithOutput(progressOut))
	if len(downloadArtfs) != 0 {
		if installer.runtime.msgHandler != nil {
			installer.runtime.msgHandler.DownloadStarting()
		}

		if len(downloadArtfs) > 0 {
			archives, err := installer.runtimeDownloader.Download(downloadArtfs, runtimeAssembler, downloadProgress)
			if err != nil {
				downloadProgress.Cancel()
				downloadProgress.Close()
				return nil, false, err
			}

			for k, v := range archives {
				unpackArchives[k] = v
			}
		}
	}
	downloadProgress.Close()

	if installer.runtime.msgHandler != nil {
		installer.runtime.msgHandler.InstallStarting()
	}

	installProgress := progress.New(mpb.WithOutput(progressOut))
	err = installer.InstallFromArchives(unpackArchives, runtimeAssembler, installProgress)
	if err != nil {
		installProgress.Cancel()
		installProgress.Close()
		return nil, false, err
	}
	installProgress.Close()

	// We still want to run PostInstall because even though no new artifact might be downloaded we still might be
	// deleting some already cached ones
	err = runtimeAssembler.PostInstall()
	if err != nil {
		return nil, false, errs.Wrap(err, "error during post installation step")
	}

	err = installer.runtime.StoreBuildEngine(runtimeAssembler.BuildEngine())
	if err != nil {
		return nil, false, locale.WrapError(err, "installer_store_build_engine_err", "Failed to store build engine value.")
	}

	err = installer.runtime.MarkInstallationComplete()
	if err != nil {
		return nil, false, errs.Wrap(err, "error marking installation as complete")
	}

	return runtimeAssembler, true, nil
}

// validateCheckpoint tries to see if the checkpoint has any chance of succeeding
func (installer *Installer) validateCheckpoint() error {
	if installer.runtime.commitID == "" {
		return &ErrNoCommit{locale.NewInputError("installer_err_runtime_no_commitid")}
	}

	checkpoint, _, err := model.FetchCheckpointForCommit(installer.runtime.commitID)
	if err != nil {
		return err
	}

	for _, change := range checkpoint {
		if model.NamespaceMatch(change.Namespace, model.NamespacePrePlatformMatch) {
			return &ErrPrePlatform{locale.NewInputError("installer_err_runtime_preplatform")}
		}
	}

	return nil
}

// InstallFromArchives will unpack the installer archive, locate the install script, and then use the installer
// script to install a runtime to the configured runtime dir. Any failures during this process will result in a
// failed installation and the install-dir being removed.
func (installer *Installer) InstallFromArchives(archives map[string]*HeadChefArtifact, a Assembler, pg *progress.Progress) error {
	var bar *progress.TotalBar
	if len(archives) > 0 {
		bar = pg.AddTotalBar(locale.T("installing"), len(archives))
	}

	err := a.PreInstall()
	if err != nil {
		pg.Cancel()
		return err
	}

	for archivePath, artf := range archives {
		if err := installer.InstallFromArchive(archivePath, artf, a, pg); err != nil {
			pg.Cancel()
			return err
		}
		bar.Increment()
	}

	return nil
}

// InstallFromArchive will unpack artifact and install it
func (installer *Installer) InstallFromArchive(archivePath string, artf *HeadChefArtifact, a Assembler, progress *progress.Progress) error {

	err := a.PreUnpackArtifact(artf)
	if err != nil {
		return err
	}

	installDir := installer.runtime.runtimeDir
	tmpRuntimeDir, upb, err := installer.unpackArchive(a.Unarchiver(), archivePath, installDir, progress)
	if err != nil {
		removeInstallDir(installDir)
		return err
	}

	err = a.PostUnpackArtifact(artf, tmpRuntimeDir, archivePath, func() { upb.Increment() })
	if err != nil {
		removeInstallDir(installDir)
		return err
	}
	upb.Complete()

	return nil
}

func (installer *Installer) unpackArchive(ua unarchiver.Unarchiver, archivePath string, installDir string, p *progress.Progress) (string, *progress.UnpackBar, error) {
	if err := installer.validateArchive(ua, archivePath); err != nil {
		return "", nil, err
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
		return tmpRuntimeDir, nil, &ErrArchiveInvalid{locale.WrapError(err, "err_unpack_prepare", "Could not prepare archive directory")}

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
		return tmpRuntimeDir, nil, &ErrArchiveInvalid{locale.WrapError(err, "err_unpack", "Unarchiving failed")}
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
func (installer *Installer) validateArchive(ua unarchiver.Unarchiver, archivePath string) error {
	if !fileutils.FileExists(archivePath) {
		return &ErrArchiveInvalid{locale.NewError("installer_err_archive_notfound", "", archivePath)}
	} else if err := ua.CheckExt(archivePath); err != nil {
		return &ErrArchiveInvalid{locale.WrapError(err, "installer_err_archive_badext", "", archivePath)}
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
