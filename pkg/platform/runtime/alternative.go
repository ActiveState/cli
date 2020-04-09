package runtime

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
)

var _ Assembler = &AlternativeRuntime{}

// AlternativeRuntime holds all the meta-data necessary to activate a runtime
// environment for an Alternative build
type AlternativeRuntime struct {
	cacheDir       string
	recipeID       strfmt.UUID
	artifactMap    map[string]*HeadChefArtifact
	artifactOrder  map[string]int
	tempInstallDir string
}

// NewAlternativeRuntime returns a new alternative runtime assembler
// It filters the provided artifact list for useable artifacts
// The recipeID is needed to define the installation directory
func NewAlternativeRuntime(artifacts []*HeadChefArtifact, cacheDir string, recipeID strfmt.UUID) (*AlternativeRuntime, *failures.Failure) {

	artifactMap := map[string]*HeadChefArtifact{}
	artifactOrder := map[string]int{}

	ar := &AlternativeRuntime{
		cacheDir: cacheDir,
		recipeID: recipeID,
	}
	for i, artf := range artifacts {

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
		downloadDir := ar.downloadDirectory(artf)

		artifactMap[downloadDir] = artf
		artifactOrder[artf.ArtifactID.String()] = i
	}

	if len(artifactMap) == 0 {
		return ar, FailNoValidArtifact.New(locale.T("err_no_valid_artifact"))
	}

	ar.artifactMap = artifactMap
	ar.artifactOrder = artifactOrder
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

func (ar *AlternativeRuntime) cachedArtifact(downloadDir string) *string {
	candidate := filepath.Join(downloadDir, constants.ArtifactArchiveName)
	if fileutils.FileExists(candidate) {
		return &candidate
	}

	return nil
}

// ArtifactsToDownloadAndUnpack returns the artifacts that we need to download
// for this project.
// It returns nothing if the final installation directory is non-empty.
// Otherwise: It filters out artifacts that have been downloaded before, and adds them to
// the list of artifacts that need to be unpacked only.
func (ar *AlternativeRuntime) ArtifactsToDownloadAndUnpack() ([]*HeadChefArtifact, map[string]*HeadChefArtifact) {
	downloadArtfs := []*HeadChefArtifact{}
	archives := map[string]*HeadChefArtifact{}

	// if final installation directory exists -> no need to download or unpack anything
	if fileutils.DirExists(ar.installationDirectory()) {
		return downloadArtfs, archives
	}

	for downloadDir, artf := range ar.artifactMap {
		cached := ar.cachedArtifact(downloadDir)
		if cached == nil {
			downloadArtfs = append(downloadArtfs, artf)
		} else {
			archives[*cached] = artf
		}
	}
	return downloadArtfs, archives
}

func (ar *AlternativeRuntime) downloadDirectory(artf *HeadChefArtifact) string {
	return filepath.Join(ar.cacheDir, "artifacts", hash.ShortHash(artf.ArtifactID.String()))
}

// DownloadDirectory returns the local directory where the artifact files should
// be downloaded to
func (ar *AlternativeRuntime) DownloadDirectory(artf *HeadChefArtifact) (string, *failures.Failure) {
	p := ar.downloadDirectory(artf)
	fail := fileutils.MkdirUnlessExists(p)
	return p, fail
}

func (ar *AlternativeRuntime) installationDirectory() string {
	finstDir := filepath.Join(ar.cacheDir, hash.ShortHash(ar.recipeID.String()))
	return finstDir
}

// InstallationDirectory returns the local directory where the artifact files
// should be unpacked to.
// For alternative build artifacts, all artifacts are unpacked into the same
// directory eventually.
func (ar *AlternativeRuntime) InstallationDirectory(_ *HeadChefArtifact) string {
	return ar.installationDirectory()
}

// PreInstall ensures that the final installation directory exists, and is useable
func (ar *AlternativeRuntime) PreInstall() *failures.Failure {
	installDir := ar.installationDirectory()

	if fileutils.FileExists(installDir) {
		// install-dir exists, but is a regular file
		return FailInstallDirInvalid.New("installer_err_installdir_isfile", installDir)
	}

	if fail := fileutils.MkdirUnlessExists(installDir); fail != nil {
		return fail
	}

	if isEmpty, fail := fileutils.IsEmptyDir(installDir); fail != nil || !isEmpty {
		if fail != nil {
			return fail
		}
		return FailInstallDirInvalid.New("installer_err_installdir_notempty", installDir)
	}

	return nil
}

// PreUnpackArtifact does nothing
func (ar *AlternativeRuntime) PreUnpackArtifact(artf *HeadChefArtifact) *failures.Failure {
	return nil
}

// PostUnpackArtifact is called after the artifacts are unpacked
// In this steps, the artifact contents are moved to its final destination.
// This step also sets up the runtime environment variables.
func (ar *AlternativeRuntime) PostUnpackArtifact(artf *HeadChefArtifact, tmpRuntimeDir string, archivePath string, cb func()) *failures.Failure {

	// final installation target
	ft := ar.InstallationDirectory(artf)

	rt, fail := envdef.NewEnvironmentDefinition(filepath.Join(tmpRuntimeDir, constants.RuntimeDefinitionFilename))
	if fail != nil {
		return fail
	}
	rt = rt.ExpandVariables(ft)

	// move files to the final installation directory
	fail = fileutils.MoveAllFilesRecursively(
		filepath.Join(tmpRuntimeDir, rt.InstallDir),
		ft, cb)
	if fail != nil {
		return fail
	}

	// move the runtime.json to the runtime environment directory
	artifactIndex, ok := ar.artifactOrder[artf.ArtifactID.String()]
	if !ok {
		logging.Error("Could not write runtime.json: artifact order for %s unknown", artf.ArtifactID.String())
		return failures.FailRuntime.New("runtime_alternative_failed_artifact_order")
	}

	fail = fileutils.MkdirUnlessExists(ar.runtimeEnvBaseDir())
	if fail != nil {
		return fail
	}

	// copy the runtime environment file to the installation directory.
	// The file name is based on the artifact order index, such that we can
	// ensure the environment definition files can be read in the correct order.
	err := rt.WriteFile(filepath.Join(ar.runtimeEnvBaseDir(), fmt.Sprintf("%06d.json", artifactIndex)))
	if err != nil {
		return failures.FailRuntime.Wrap(err, "runtime_alternative_failed_destination", ar.runtimeEnvBaseDir())
	}

	if err := os.RemoveAll(tmpRuntimeDir); err != nil {
		logging.Error("removing %s after unpacking runtime: %v", tmpRuntimeDir, err)
	}
	return nil
}

func (ar *AlternativeRuntime) runtimeEnvBaseDir() string {
	return filepath.Join(ar.installationDirectory(), constants.LocalRuntimeEnvironmentDirectory)
}

// assembleRuntimeDefinition assembles the environment from runtime definition files copied to the
// installation directory (at runtimeEnvBaseDir())
// This function expects files named `"00001.json", "00002.json", ...` that are installed in the
// PostUnpackArtifact step.  It sorts them by name, parses them and merges the EnvironmentDefinition
//
// As an optimization step, the merged environment definition is cached and written back to
// `<runtimeEnvBaseDir()>/runtime.json`. If this file exits, we can just return its parsed contents and skip parsing
// the many individual runtime definition files.
func (ar *AlternativeRuntime) assembleRuntimeDefinition() (*envdef.EnvironmentDefinition, *failures.Failure) {
	mergedRuntimeDefinitionFile := filepath.Join(ar.runtimeEnvBaseDir(), constants.RuntimeDefinitionFilename)
	if fileutils.FileExists(mergedRuntimeDefinitionFile) {
		rt, fail := envdef.NewEnvironmentDefinition(mergedRuntimeDefinitionFile)
		if fail == nil {
			return rt, nil
		}
		logging.Warning("Failed to unmarshal the merged runtime definition file at %s: %v", mergedRuntimeDefinitionFile, fail.ToError())
	}

	files, err := ioutil.ReadDir(ar.runtimeEnvBaseDir())
	if err != nil {
		return nil, FailRuntimeInvalidEnvironment.New("err_no_environment_definition")
	}

	filenames := make([]string, 0, len(files))
	for _, f := range files {
		if ok, _ := regexp.MatchString("[0-9]{6}.json", f.Name()); ok {
			filenames = append(filenames, f.Name())
		}
	}
	sort.Strings(filenames)

	var rtGlobal *envdef.EnvironmentDefinition

	for _, fn := range filenames {
		rtPath := filepath.Join(ar.runtimeEnvBaseDir(), fn)
		rt, fail := envdef.NewEnvironmentDefinition(rtPath)
		if fail != nil {
			return nil, fail
		}
		if rtGlobal == nil {
			rtGlobal = rt
			continue
		}
		rtGlobal, err = rtGlobal.Merge(rt)
		if err != nil {
			logging.Warning("Failed to merge environment definition file %s: %v", rtPath, err)
			continue
		}
	}

	if rtGlobal == nil {
		return nil, FailRuntimeInvalidEnvironment.New("err_no_environment_definition")
	}

	err = rtGlobal.WriteFile(mergedRuntimeDefinitionFile)
	if err != nil {
		// It should still work without writing the merged runtime definition file
		logging.Warning(fmt.Sprintf("Failed to write merged runtime definition file at %s", mergedRuntimeDefinitionFile))
	}

	return rtGlobal, nil
}

// GetEnv returns the environment variable configuration for this build
func (ar *AlternativeRuntime) GetEnv(inherit bool, _ string) (map[string]string, *failures.Failure) {
	rt, fail := ar.assembleRuntimeDefinition()
	if fail != nil {
		return nil, fail
	}
	return rt.GetEnv(inherit), nil
}
