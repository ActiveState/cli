package store

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
)

// Store manages the storing and loading of persistable information about the runtime
type Store struct {
	installPath string
	storagePath string
}

type StoredArtifact struct {
	ArtifactID artifact.ArtifactID           `json:"artifactID"`
	Files      []string                      `json:"files"`
	Dirs       []string                      `json:"dirs"`
	EnvDef     *envdef.EnvironmentDefinition `json:"envdef"`
}

func NewStoredArtifact(artifactID artifact.ArtifactID, files []string, dirs []string, envDef *envdef.EnvironmentDefinition) StoredArtifact {
	return StoredArtifact{
		ArtifactID: artifactID,
		Files:      files,
		Dirs:       dirs,
		EnvDef:     envDef,
	}
}

type StoredArtifactMap = map[artifact.ArtifactID]StoredArtifact

func New(installPath string) *Store {
	return &Store{
		installPath,
		filepath.Join(installPath, constants.LocalRuntimeEnvironmentDirectory),
	}
}

func (s *Store) markerFile() string {
	return filepath.Join(s.storagePath, constants.RuntimeInstallationCompleteMarker)
}

func (s *Store) buildEngineFile() string {
	return filepath.Join(s.storagePath, constants.RuntimeBuildEngineStore)
}

func (s *Store) recipeFile() string {
	return filepath.Join(s.storagePath, constants.RuntimeRecipeStore)
}

func (s *Store) HasMarker() bool {
	if fileutils.FileExists(s.markerFile()) {
		return true
	}
	return false
}

// MatchesCommit checks if stored runtime is complete and can be loaded
func (s *Store) MatchesCommit(commitID strfmt.UUID) bool {
	marker := s.markerFile()
	if !fileutils.FileExists(marker) {
		logging.Debug("Marker does not exist: %s", marker)
		return false
	}

	contents, err := fileutils.ReadFile(marker)
	if err != nil {
		logging.Error("Could not read marker: %v", err)
		return false
	}

	logging.Debug("MatchesCommit for %s, %s==%s", marker, string(contents), commitID.String())
	return strings.TrimSpace(string(contents)) == commitID.String()
}

// MarkInstallationComplete writes the installation complete marker to the runtime directory
func (s *Store) MarkInstallationComplete(commitID strfmt.UUID) error {
	markerFile := s.markerFile()
	markerDir := filepath.Dir(markerFile)
	err := fileutils.MkdirUnlessExists(markerDir)
	if err != nil {
		return errs.Wrap(err, "could not create completion marker directory")
	}
	err = fileutils.WriteFile(markerFile, []byte(commitID.String()))
	if err != nil {
		return errs.Wrap(err, "could not set completion marker")
	}
	return nil
}

// BuildEngine returns the runtime build engine value stored in the runtime directory
func (s *Store) BuildEngine() (model.BuildEngine, error) {
	storeFile := s.buildEngineFile()

	data, err := fileutils.ReadFile(storeFile)
	if err != nil {
		return model.UnknownEngine, errs.Wrap(err, "Could not read build engine cache store.")
	}

	return model.ParseBuildEngine(string(data)), nil
}

// StoreBuildEngine stores the build engine value in the runtime directory
func (s *Store) StoreBuildEngine(buildEngine model.BuildEngine) error {
	storeFile := s.buildEngineFile()
	storeDir := filepath.Dir(storeFile)
	logging.Debug("Storing build engine %s at %s", buildEngine.String(), storeFile)
	err := fileutils.MkdirUnlessExists(storeDir)
	if err != nil {
		return errs.Wrap(err, "Could not create completion marker directory.")
	}
	err = fileutils.WriteFile(storeFile, []byte(buildEngine.String()))
	if err != nil {
		return errs.Wrap(err, "Could not store build engine string.")
	}
	return nil
}

// Recipe returns the recipe the stored runtime has been built with
func (s *Store) Recipe() (*inventory_models.Recipe, error) {
	data, err := fileutils.ReadFile(s.recipeFile())
	if err != nil {
		return nil, errs.Wrap(err, "Could not read recipe file.")
	}

	var recipe inventory_models.Recipe
	err = json.Unmarshal(data, &recipe)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse recipe file.")
	}
	return &recipe, err
}

// StoreRecipe stores a along side the stored runtime
func (s *Store) StoreRecipe(recipe *inventory_models.Recipe) error {
	data, err := json.Marshal(recipe)
	if err != nil {
		return errs.Wrap(err, "Could not marshal recipe.")
	}
	err = fileutils.WriteFile(s.recipeFile(), data)
	if err != nil {
		return errs.Wrap(err, "Could not write recipe file.")
	}
	return nil
}

// Artifacts loads artifact information collected during the installation.
// It includes the environment definition configuration and files installed for this artifact.
func (s *Store) Artifacts() (StoredArtifactMap, error) {
	stored := make(StoredArtifactMap)
	jsonDir := filepath.Join(s.storagePath, constants.ArtifactMetaDir)
	if !fileutils.DirExists(jsonDir) {
		return stored, nil
	}

	files, err := ioutil.ReadDir(jsonDir)
	if err != nil {
		return stored, errs.Wrap(err, "Readdir %s failed", jsonDir)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		var artifactStore StoredArtifact
		jsonBlob, err := fileutils.ReadFile(filepath.Join(jsonDir, file.Name()))
		if err != nil {
			return stored, errs.Wrap(err, "Could not read artifact meta file")
		}
		if err := json.Unmarshal(jsonBlob, &artifactStore); err != nil {
			return stored, errs.Wrap(err, "Could not unmarshal artifact meta file")
		}

		stored[artifactStore.ArtifactID] = artifactStore
	}

	return stored, nil
}

// DeleteArtifactStore deletes the stored information for a specific artifact from the store
func (s *Store) DeleteArtifactStore(id artifact.ArtifactID) error {
	jsonFile := filepath.Join(s.storagePath, constants.ArtifactMetaDir, id.String()+".json")
	if !fileutils.FileExists(jsonFile) {
		return nil
	}
	return os.Remove(jsonFile)
}

func (s *Store) StoreArtifact(artf StoredArtifact) error {
	// Save artifact cache information
	jsonBlob, err := json.Marshal(artf)
	if err != nil {
		return errs.Wrap(err, "Failed to marshal artifact cache information")
	}
	jsonFile := filepath.Join(s.storagePath, constants.ArtifactMetaDir, artf.ArtifactID.String()+".json")
	if err := fileutils.WriteFile(jsonFile, jsonBlob); err != nil {
		return errs.Wrap(err, "Failed to write artifact cache information")
	}
	return nil
}

func (s *Store) EnvDef() (*envdef.EnvironmentDefinition, error) {
	mergedRuntimeDefinitionFile := filepath.Join(s.storagePath, constants.RuntimeDefinitionFilename)
	envDef, err := envdef.NewEnvironmentDefinition(mergedRuntimeDefinitionFile)
	if err != nil {
		return nil, locale.WrapError(
			err, "err_no_environment_definition",
			"Your installation seems corrupted.\nPlease try to re-run this command, as it may fix the problem.  If the problem persists, please report it in our forum: {{.V0}}",
			constants.ForumsURL,
		)
	}
	return envDef, nil
}

func (s *Store) Environ(inherit bool) (map[string]string, error) {
	envDef, err := s.EnvDef()
	if err != nil {
		return nil, errs.Wrap(err, "Could not grab EnvDef")
	}
	return envDef.GetEnv(inherit), nil
}

func (s *Store) UpdateEnviron(orderedArtifacts []artifact.ArtifactID) (*envdef.EnvironmentDefinition, error) {
	artifacts, err := s.Artifacts()
	if err != nil {
		return nil, errs.Wrap(err, "Could not retrieve stored artifacts")
	}

	rtGlobal, err := s.updateEnviron(orderedArtifacts, artifacts)
	if err != nil {
		return nil, err
	}

	return rtGlobal, rtGlobal.WriteFile(filepath.Join(s.storagePath, constants.RuntimeDefinitionFilename))
}

func (s *Store) updateEnviron(orderedArtifacts []artifact.ArtifactID, artifacts StoredArtifactMap) (*envdef.EnvironmentDefinition, error) {
	var rtGlobal *envdef.EnvironmentDefinition
	// use artifact order as returned by the build status response form the HC for merging artifacts
	for _, artID := range orderedArtifacts {
		a, ok := artifacts[artID]
		if !ok {
			continue
		}

		if rtGlobal == nil {
			rtGlobal = a.EnvDef
			continue
		}
		var err error
		rtGlobal, err = rtGlobal.Merge(a.EnvDef)
		if err != nil {
			return nil, errs.Wrap(err, "Could not merge envdef")
		}
	}

	return rtGlobal, nil
}

// InstallPath returns the installation path of the runtime
func (s *Store) InstallPath() string {
	return s.installPath
}
