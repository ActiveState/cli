package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
)

var _ Assembler = &AlternativeRuntime{}

// AlternativeRuntime holds all the meta-data necessary to activate a runtime
// environment for an Alternative build
type AlternativeRuntime struct {
	runtimeDir         string
	recipeID           strfmt.UUID
	artifactsRequested []*HeadChefArtifact
	tempInstallDir     string
	cache              []artifactCacheMeta
}

type artifactCacheMeta struct {
	ArtifactID strfmt.UUID
	Files      []string
}

// NewAlternativeRuntime returns a new alternative runtime assembler
// It filters the provided artifact list for useable artifacts
// The recipeID is needed to define the installation directory
func NewAlternativeRuntime(artifacts []*HeadChefArtifact, cacheDir string, recipeID strfmt.UUID) (*AlternativeRuntime, *failures.Failure) {
	if rtutils.BuiltViaCI {
		return nil, failures.FailRuntime.New("Alternative builds are not yet supported. Please contact support if you are seeing this error.")
	}

	ar := &AlternativeRuntime{
		runtimeDir: cacheDir,
		recipeID:   recipeID,
	}
	for _, artf := range artifacts {

		if artf.URI == "" {
			continue
		}
		filename := filepath.Base(artf.URI.String())
		if !strings.HasSuffix(filename, ar.InstallerExtension()) {
			continue
		}

		// For now we are excluding terminal artifacts ie., the artifacts that a packaging step would produce.
		// Right now, these artifacts are empty anyways...
		if artf.IngredientVersionID == "" {
			continue
		}

		ar.artifactsRequested = append(ar.artifactsRequested, artf)
	}

	if len(ar.artifactsRequested) == 0 {
		return ar, FailNoValidArtifact.New(locale.T("err_no_valid_artifact"))
	}

	var err error
	ar.cache, err = ar.artifactCache()
	if err != nil {
		return ar, FailNoValidArtifact.Wrap(err, "Could not grab artifact cache")
	}

	return ar, nil
}

// InstallerExtension is always .tar.gz
func (ar *AlternativeRuntime) InstallerExtension() string {
	return ".tar.gz"
}

// Unarchiver always returns an unarchiver for gzipped tarballs
func (ar *AlternativeRuntime) Unarchiver() unarchiver.Unarchiver {
	return unarchiver.NewTarGz()
}

// BuildEngine always returns Alternative
func (ar *AlternativeRuntime) BuildEngine() BuildEngine {
	return Alternative
}

func (ar *AlternativeRuntime) artifactCache() ([]artifactCacheMeta, error) {
	cache := []artifactCacheMeta{}
	jsonFile := filepath.Join(ar.runtimeEnvBaseDir(), constants.ArtifactCacheFileName)
	if !fileutils.FileExists(jsonFile) {
		return cache, nil
	}

	jsonBlob, fail := fileutils.ReadFile(jsonFile)
	if fail != nil {
		return cache, errs.Wrap(fail, "Could not read artifact cache file")
	}
	if err := json.Unmarshal(jsonBlob, &cache); err != nil {
		return cache, errs.Wrap(fail, "Could not unmarshal artifact cache file")
	}

	return cache, nil
}

func (ar *AlternativeRuntime) storeArtifactCache() error {
	// Save artifact cache information
	jsonBlob, err := json.Marshal(ar.cache)
	if err != nil {
		return errs.Wrap(err, "Failed to marshal artifact cache information")
	}
	jsonFile := filepath.Join(ar.runtimeEnvBaseDir(), constants.ArtifactCacheFileName)
	if fail := fileutils.WriteFile(jsonFile, jsonBlob); fail != nil {
		return errs.Wrap(fail, "Failed to write artifact cache information")
	}
	return nil
}

// ArtifactsToDownload returns the artifacts that we need to download
// for this project.
func (ar *AlternativeRuntime) ArtifactsToDownload() []*HeadChefArtifact {
	return artifactsToDownload(artifactCacheToUuids(ar.cache), ar.artifactsRequested)
}

// ArtifactsToDownload returns the artifacts that we need to download
// for this project.
func artifactsToDownload(artifactCacheUuids []strfmt.UUID, artifactsRequested []*HeadChefArtifact) []*HeadChefArtifact {
	downloadArtfs := []*HeadChefArtifact{}
	for _, v := range artifactsRequested {
		if v.ArtifactID != nil && !funk.Contains(artifactCacheUuids, *v.ArtifactID) {
			downloadArtfs = append(downloadArtfs, v)
		}
	}
	return downloadArtfs
}

// IsInstalled checks if the merged runtime environment definition file exists and whether any artifacts need to be
// downloaded or deleted
func (ar *AlternativeRuntime) IsInstalled() bool {
	download := artifactsToDownload(artifactCacheToUuids(ar.cache), ar.artifactsRequested)
	_, delete := artifactsToKeepAndDelete(ar.cache, artifactsToUuids(ar.artifactsRequested))

	// runtime environment definition file
	red := filepath.Join(ar.runtimeEnvBaseDir(), constants.RuntimeDefinitionFilename)
	return fileutils.FileExists(red) && len(download) == 0 && len(delete) == 0
}

func (ar *AlternativeRuntime) downloadDirectory(artf *HeadChefArtifact) string {
	return filepath.Join(ar.runtimeDir, constants.LocalRuntimeEnvironmentDirectory, "artifacts", hash.ShortHash(artf.ArtifactID.String()))
}

// DownloadDirectory returns the local directory where the artifact files should
// be downloaded to
func (ar *AlternativeRuntime) DownloadDirectory(artf *HeadChefArtifact) (string, *failures.Failure) {
	p := ar.downloadDirectory(artf)
	fail := fileutils.MkdirUnlessExists(p)
	return p, fail
}

// PreInstall ensures that the final installation directory exists, and is useable
func (ar *AlternativeRuntime) PreInstall() *failures.Failure {
	if fileutils.FileExists(ar.runtimeDir) {
		// install-dir exists, but is a regular file
		return FailInstallDirInvalid.New("installer_err_installdir_isfile", ar.runtimeDir)
	}

	if !fileutils.DirExists(ar.runtimeDir) {
		if fail := fileutils.Mkdir(ar.runtimeDir); fail != nil {
			return fail
		}

		// No need to delete files if this is a new dir
		return nil
	}

	// Delete artifacts that are no longer part of the request
	var delete []artifactCacheMeta
	ar.cache, delete = artifactsToKeepAndDelete(ar.cache, artifactsToUuids(ar.artifactsRequested))
	for _, v := range delete {
		for _, file := range v.Files {
			if !fileutils.TargetExists(file) {
				continue // don't care it's already deleted (might have been deleted by another artifact that supplied the same file)
			}
			if err := os.Remove(file); err != nil {
				return failures.FailIO.Wrap(err, locale.Tl("err_rm_artf", "Could not remove old package file at {{.V0}}.", file))
			}
		}
	}

	if err := ar.storeArtifactCache(); err != nil {
		return failures.FailIO.Wrap(err, locale.Tl("err_store_artf", "Could not store artifact cache."))
	}

	return nil
}

func artifactsToKeepAndDelete(artifactCache []artifactCacheMeta, artifactRequestUuids []strfmt.UUID) (keep []artifactCacheMeta, delete []artifactCacheMeta) {
	keep = []artifactCacheMeta{}
	delete = []artifactCacheMeta{}
	for _, v := range artifactCache {
		if funk.Contains(artifactRequestUuids, v.ArtifactID) {
			keep = append(keep, v)
			continue
		}
		delete = append(delete, v)
	}
	return keep, delete
}

// PreUnpackArtifact does nothing
func (ar *AlternativeRuntime) PreUnpackArtifact(artf *HeadChefArtifact) *failures.Failure {
	return nil
}

// PostUnpackArtifact is called after the artifacts are unpacked
// In this steps, the artifact contents are moved to its final destination.
// This step also sets up the runtime environment variables.
func (ar *AlternativeRuntime) PostUnpackArtifact(artf *HeadChefArtifact, tmpRuntimeDir string, archivePath string, cb func()) *failures.Failure {
	envDef, fail := envdef.NewEnvironmentDefinition(filepath.Join(tmpRuntimeDir, constants.RuntimeDefinitionFilename))
	if fail != nil {
		return fail
	}
	constants := envdef.NewConstants(ar.runtimeDir)
	envDef = envDef.ExpandVariables(constants)
	err := envDef.ApplyFileTransforms(tmpRuntimeDir, constants)
	if err != nil {
		return failures.FailRuntime.Wrap(err, locale.Tl("runtime_alternative_file_transforms_err", "Could not apply necessary file transformations after unpacking"))
	}

	artMeta := artifactCacheMeta{*artf.ArtifactID, []string{}}
	onMoveFile := func(fromPath, toPath string) {
		if fileutils.IsDir(toPath) {
			artMeta.Files = append(artMeta.Files, fileutils.ListDir(toPath, false)...)
		} else {
			artMeta.Files = append(artMeta.Files, toPath)
		}
		cb()
	}

	// move files to the final installation directory
	fail = fileutils.MoveAllFilesRecursively(
		filepath.Join(tmpRuntimeDir, envDef.InstallDir),
		ar.runtimeDir, onMoveFile)
	if fail != nil {
		return fail
	}

	ar.cache = append(ar.cache, artMeta)

	// move the runtime.json to the runtime environment directory
	artifactIndex := funk.IndexOf(ar.artifactsRequested, artf)
	if artifactIndex == -1 {
		logging.Error("Could not write runtime.json: artifact order for %s unknown", artf.ArtifactID.String())
		return failures.FailRuntime.New(locale.Tl("runtime_alternative_failed_artifact_order", "Could not write runtime.json file: internal error"))
	}

	fail = fileutils.MkdirUnlessExists(ar.runtimeEnvBaseDir())
	if fail != nil {
		return fail
	}

	// copy the runtime environment file to the installation directory.
	// The file name is based on the artifact order index, such that we can
	// ensure the environment definition files can be read in the correct order.
	err = envDef.WriteFile(filepath.Join(ar.runtimeEnvBaseDir(), fmt.Sprintf("%06d.json", artifactIndex)))
	if err != nil {
		return failures.FailRuntime.Wrap(err, locale.Tl("runtime_alternative_failed_destination", "Installation failed due to to failed write to directory {{.V0}}", ar.runtimeEnvBaseDir()))
	}

	if err := os.RemoveAll(tmpRuntimeDir); err != nil {
		logging.Error("removing tmpdir %s after unpacking runtime: %v", tmpRuntimeDir, err)
	}
	if err := os.Remove(archivePath); err != nil {
		logging.Error("removing archive %s after unpacking runtime: %v", archivePath, err)
	}
	return nil
}

func (ar *AlternativeRuntime) runtimeEnvBaseDir() string {
	return filepath.Join(ar.runtimeDir, constants.LocalRuntimeEnvironmentDirectory)
}

// PostInstall merges all runtime environment definition files for the artifacts in order
// This function expects files named `"00001.json", "00002.json", ...` that are installed in the
// PostUnpackArtifact step.  It sorts them by name, parses them and merges the EnvironmentDefinition
//
// The merged environment definition is cached and written back to `<runtimeEnvBaseDir()>/runtime.json`.
// This file also serves as a marker that the installation was successfully completed.
//
// It also checks if a PPM shim needs to be installed
func (ar *AlternativeRuntime) PostInstall() error {
	mergedRuntimeDefinitionFile := filepath.Join(ar.runtimeEnvBaseDir(), constants.RuntimeDefinitionFilename)
	var rtGlobal *envdef.EnvironmentDefinition

	files, err := ioutil.ReadDir(ar.runtimeEnvBaseDir())
	if err != nil {
		return errs.Wrap(err, "Could not find the runtime environment directory")
	}

	filenames := make([]string, 0, len(files))
	for _, f := range files {
		if ok, _ := regexp.MatchString("[0-9]{6}.json", f.Name()); ok {
			filenames = append(filenames, f.Name())
		}
	}
	sort.Strings(filenames)
	for _, fn := range filenames {
		rtPath := filepath.Join(ar.runtimeEnvBaseDir(), fn)
		rt, fail := envdef.NewEnvironmentDefinition(rtPath)
		if fail != nil {
			return errs.Wrap(fail, "Failed to parse runtime environment definition file at %s", rtPath)
		}
		if rtGlobal == nil {
			rtGlobal = rt
			continue
		}
		rtGlobal, err = rtGlobal.Merge(rt)
		if err != nil {
			return errs.Wrap(err, "Failed merge environment definitions")
		}
	}

	if rtGlobal == nil {
		return errs.New("No runtime environment definition file at %s", ar.runtimeDir)
	}

	if activePerlPath := rtGlobal.FindBinPathFor(constants.ActivePerlExecutable); activePerlPath != "" {
		err = installPPMShim(activePerlPath)
		if err != nil {
			return errs.Wrap(err, "Failed to install the PPM shim command at %s", activePerlPath)
		}
	}

	err = rtGlobal.WriteFile(mergedRuntimeDefinitionFile)
	if err != nil {
		return errs.Wrap(err, "Failed to write merged runtime definition file at %s", mergedRuntimeDefinitionFile)
	}

	if err := ar.storeArtifactCache(); err != nil {
		return errs.Wrap(err, "Could not store artifact cache")
	}

	return nil
}

// GetEnv returns the environment variable configuration for this build
func (ar *AlternativeRuntime) GetEnv(inherit bool, _ string) (map[string]string, error) {
	mergedRuntimeDefinitionFile := filepath.Join(ar.runtimeEnvBaseDir(), constants.RuntimeDefinitionFilename)
	rt, fail := envdef.NewEnvironmentDefinition(mergedRuntimeDefinitionFile)
	if fail != nil {
		return nil, locale.WrapError(
			fail, "err_no_environment_definition",
			"Your installation seems corrupted.\nPlease try to re-run this command, as it may fix the problem.  If the problem persists, please report it in our forum: {{.V0}}",
			constants.ForumsURL,
		)
	}
	return rt.GetEnv(inherit), nil
}
