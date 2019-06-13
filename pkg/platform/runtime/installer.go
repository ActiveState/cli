package runtime

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

var (
	// FailInstallDirInvalid represents a Failure due to the working-dir for an installation being invalid in some way.
	FailInstallDirInvalid = failures.Type("runtime.installdir.invalid", failures.FailIO)

	// FailArchiveInvalid represents a Failure due to the installer archive file being invalid in some way.
	FailArchiveInvalid = failures.Type("runtime.archive.invalid", failures.FailIO)

	// FailArchiveNoInstallDir represents a Failure due to an archive not having an install dir
	FailArchiveNoInstallDir = failures.Type("runtime.archive.noinstalldir", FailArchiveInvalid)

	// FailRuntimeInvalid represents a Failure due to a runtime being invalid in some way prior to installation.
	FailRuntimeInvalid = failures.Type("runtime.runtime.invalid", failures.FailIO)

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
	downloadDir       string
	cacheDir          string
	installDirs       []string
	runtimeDownloader Downloader
	onDownload        func()
	onInstall         func()
	archiver          archiver.Archiver
	unarchiver        archiver.Unarchiver
}

// InitInstaller creates a new RuntimeInstaller
func InitInstaller() (*Installer, *failures.Failure) {
	downloadDir, err := ioutil.TempDir("", "state-runtime-downloader")
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}
	return NewInstaller(downloadDir, config.CachePath(), InitDownload(downloadDir))
}

// NewInstaller creates a new RuntimeInstaller after verifying the provided install-dir
// exists as a directory or can be created.
func NewInstaller(downloadDir string, cacheDir string, downloader Downloader) (*Installer, *failures.Failure) {
	installer := &Installer{
		downloadDir:       downloadDir,
		cacheDir:          cacheDir,
		runtimeDownloader: downloader,
		archiver:          Archiver(),
		unarchiver:        Unarchiver(),
	}

	return installer, nil
}

// Install will download the installer archive and invoke InstallFromArchive
func (installer *Installer) Install() *failures.Failure {
	if fail := installer.validateCheckpoint(); fail != nil {
		return fail
	}

	artifactMap, fail := installer.fetchArtifactMap()
	if fail != nil {
		return fail
	}

	downloadURLs := []*url.URL{}
	for url, installDir := range artifactMap {
		if !fileutils.DirExists(installDir) {
			downloadURLs = append(downloadURLs, url)
		}
	}

	if len(downloadURLs) == 0 {
		// Already installed, no need to download or install
		logging.Debug("Nothing to download")
		return nil
	}

	if installer.onDownload != nil {
		installer.onDownload()
	}

	archives := []string{}

	filenames, fail := installer.runtimeDownloader.Download(downloadURLs)
	if fail != nil {
		return fail
	}

	for _, filename := range filenames {
		archives = append(archives, filepath.Join(installer.downloadDir, filename))
	}

	return installer.InstallFromArchives(archives)
}

// validateCheckpoint tries to see if the checkpoint has any chance of succeeding
func (installer *Installer) validateCheckpoint() *failures.Failure {
	pj := project.Get()
	if pj.CommitID() == "" {
		return FailNoCommits.New("installer_err_runtime_no_commits", model.ProjectURL(pj.Owner(), pj.Name(), ""))
	}

	checkpoint, fail := model.FetchCheckpointForCommit(strfmt.UUID(pj.CommitID()))
	if fail != nil {
		return fail
	}

	for _, change := range checkpoint {
		if model.NamespaceMatch(change.Namespace, model.NamespacePrePlatform) {
			return FailPrePlatformNotSupported.New("installer_err_runtime_preplatform")
		}
	}

	return nil
}

func (installer *Installer) fetchArtifactMap() (map[*url.URL]string, *failures.Failure) {
	artifactMap := map[*url.URL]string{}

	artifactURLs, fail := installer.runtimeDownloader.FetchArtifactURLs()
	if fail != nil {
		return artifactMap, fail
	}

	for _, url := range artifactURLs {
		installDir, fail := installer.installDir(filepath.Base(url.Path))
		if fail != nil {
			return artifactMap, fail
		}
		artifactMap[url] = installDir
		installer.installDirs = append(installer.installDirs, installDir)
	}

	return artifactMap, nil
}

// InstallFromArchives will unpack the installer archive, locate the install script, and then use the installer
// script to install a runtime to the configured runtime dir. Any failures during this process will result in a
// failed installation and the install-dir being removed.
func (installer *Installer) InstallFromArchives(archivePaths []string) *failures.Failure {
	if installer.onInstall != nil {
		installer.onInstall()
	}

	for _, archivePath := range archivePaths {
		if fail := installer.InstallFromArchive(archivePath); fail != nil {
			return fail
		}
	}

	return nil
}

// InstallFromArchive will unpack artifact and install it
func (installer *Installer) InstallFromArchive(archivePath string) *failures.Failure {
	var installDir string
	var fail *failures.Failure
	if installDir, fail = installer.installDir(filepath.Base(archivePath)); fail != nil {
		return fail
	}
	installer.installDirs = append(installer.installDirs, installDir)

	if fail := fileutils.MkdirUnlessExists(installDir); fail != nil {
		return fail
	}

	if fail := installer.unpackArchive(archivePath, installDir); fail != nil {
		removeInstallDir(installDir)
		return fail
	}

	metaData, fail := InitMetaData(installDir)
	if fail != nil {
		removeInstallDir(installDir)
		return fail
	}

	if fail = installer.Relocate(metaData); fail != nil {
		removeInstallDir(installDir)
		return fail
	}

	return nil
}

// InstallDirs returns all the artifact install paths required by the current configuration.
// WARNING: This will always return an empty slice UNLESS Install() or InstallFromArchive() was called!
func (installer *Installer) InstallDirs() []string {
	return funk.Uniq(installer.installDirs).([]string)
}

func (installer *Installer) installDir(filename string) (string, *failures.Failure) {
	installDir := filepath.Join(installer.cacheDir, shortHash(filename))

	if fileutils.FileExists(installDir) {
		// install-dir exists, but is a regular file
		return "", FailInstallDirInvalid.New("installer_err_installdir_isfile", installDir)
	}

	return installDir, nil
}

func (installer *Installer) unpackArchive(archivePath string, installDir string) *failures.Failure {
	if isEmpty, fail := fileutils.IsEmptyDir(installDir); fail != nil || !isEmpty {
		if fail != nil {
			return fail
		}
		return FailRuntimeInstallation.New("installer_err_installdir_notempty", installDir)
	}

	if fail := installer.validateArchive(archivePath); fail != nil {
		return fail
	}

	tmpRuntimeDir := filepath.Join(installDir, uuid.New().String())
	archiveName := strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath))

	// the above only strips .gz, so account for .tar.gz use-case
	// it's fine to run this on windows cause those files won't end in .tar anyway
	archiveName = strings.TrimSuffix(archiveName, ".tar")

	logging.Debug("Unarchiving %s", archivePath)
	err := installer.unarchiver.Unarchive(archivePath, tmpRuntimeDir)
	logging.Debug("Done")
	if err != nil {
		return FailArchiveInvalid.Wrap(err)
	}

	// Detect the install dir
	tmpInstallDir := ""
	installDirs := strings.Split(constants.RuntimeInstallDirs, ",")
	for _, dir := range installDirs {
		currentDir := filepath.Join(tmpRuntimeDir, archiveName, dir)
		if fileutils.DirExists(currentDir) {
			tmpInstallDir = currentDir
		}
	}
	if tmpInstallDir == "" {
		return FailArchiveNoInstallDir.New("installer_err_runtime_missing_install_dir", tmpRuntimeDir, constants.RuntimeInstallDirs)
	}

	if fail := fileutils.MoveAllFiles(tmpInstallDir, installDir); fail != nil {
		logging.Error("moving files from %s after unpacking runtime: %v", tmpInstallDir, fail.ToError())
		return FailRuntimeInstallation.New("installer_err_runtime_move_files_failed", tmpInstallDir)
	}

	tmpMetaFile := filepath.Join(tmpRuntimeDir, archiveName, constants.RuntimeMetaFile)
	if fileutils.FileExists(tmpMetaFile) {
		target := filepath.Join(installDir, constants.RuntimeMetaFile)
		if fail := fileutils.MkdirUnlessExists(filepath.Dir(target)); fail != nil {
			return fail
		}
		if err := os.Rename(tmpMetaFile, target); err != nil {
			return FailRuntimeInstallation.Wrap(err)
		}
	}

	if err = os.RemoveAll(tmpRuntimeDir); err != nil {
		logging.Error("removing %s after unpacking runtime: %v", tmpRuntimeDir, err)
		return FailRuntimeInstallation.New("installer_err_runtime_rm_installdir", tmpRuntimeDir)
	}

	return nil
}

// OnDownload registers a function to be called when a download occurs
func (installer *Installer) OnDownload(f func()) {
	installer.onDownload = f
}

// OnInstall registers a function to be called when an install occurs
func (installer *Installer) OnInstall(f func()) {
	installer.onInstall = f
}

// Relocate will look through all of the files in this installation and replace any
// character sequence in those files containing the given prefix.
func (installer *Installer) Relocate(metaData *MetaData) *failures.Failure {
	prefix := metaData.RelocationDir

	if len(prefix) == 0 || prefix == metaData.Path {
		return nil
	}

	logging.Debug("relocating '%s' to '%s'", prefix, metaData.Path)
	binariesSeparate := runtime.GOOS == "linux" && metaData.RelocationTargetBinaries != ""

	// Replace plain text files
	err := fileutils.ReplaceAllInDirectory(metaData.Path, prefix, metaData.Path,
		// Check if we want to include this
		func(p string, contents []byte) bool {
			return !strings.HasSuffix(p, constants.RuntimeMetaFile) && (!binariesSeparate || !fileutils.IsBinary(contents))
		})
	if err != nil {
		return FailRuntimeInstallation.Wrap(err)
	}

	if binariesSeparate {
		replacement := filepath.Join(metaData.Path, metaData.RelocationTargetBinaries)
		// Replace binary files
		err = fileutils.ReplaceAllInDirectory(metaData.Path, prefix, replacement,
			// Binaries only
			func(p string, contents []byte) bool { return fileutils.IsBinary(contents) })

		if err != nil {
			return FailRuntimeInstallation.Wrap(err)
		}
	}

	logging.Debug("Done")

	return nil
}

// validateArchive ensures the given path to archive is an actual file and that its suffix is a well-known
// suffix for tar+gz files.
func (installer *Installer) validateArchive(archivePath string) *failures.Failure {
	if !fileutils.FileExists(archivePath) {
		return FailArchiveInvalid.New("installer_err_archive_notfound", archivePath)
	} else if installer.archiver.CheckExt(archivePath) != nil {
		return FailArchiveInvalid.New("installer_err_archive_badext", archivePath)
	}
	return nil
}

func removeInstallDir(installDir string) {
	if err := os.RemoveAll(installDir); err != nil {
		logging.Errorf("attempting to remove install dir '%s': %v", installDir, err)
	}
}

// shortHash will return the first 4 bytes in base16 of the sha1 sum of the provided data.
//
// For example:
//   shortHash("ActiveState-TestProject-python2")
// 	 => e784c7e0
//
// This is useful for creating a shortened namespace for language installations.
func shortHash(data ...string) string {
	h := sha1.New()
	io.WriteString(h, strings.Join(data, ""))
	return fmt.Sprintf("%x", h.Sum(nil)[:4])
}
