package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/go-openapi/strfmt"
)

// Store manages the storing and loading of persistable information about the runtime
type Store struct {
	installPath string
	storagePath string
}

type StoredArtifact struct {
	ArtifactID strfmt.UUID                   `json:"artifactID"`
	Files      []string                      `json:"files"`
	Dirs       []string                      `json:"dirs"`
	EnvDef     *envdef.EnvironmentDefinition `json:"envdef"`
}

func NewStoredArtifact(artifactID strfmt.UUID, files []string, dirs []string, envDef *envdef.EnvironmentDefinition) StoredArtifact {
	return StoredArtifact{
		ArtifactID: artifactID,
		Files:      files,
		Dirs:       dirs,
		EnvDef:     envDef,
	}
}

type StoredArtifactMap = map[strfmt.UUID]StoredArtifact

func New(installPath string) *Store {
	return &Store{
		installPath,
		filepath.Join(installPath, constants.LocalRuntimeEnvironmentDirectory),
	}
}

func (s *Store) buildEngineFile() string {
	return filepath.Join(s.storagePath, constants.RuntimeBuildEngineStore)
}

func (s *Store) recipeFile() string {
	return filepath.Join(s.storagePath, constants.RuntimeRecipeStore)
}

func (s *Store) buildPlanFile() string {
	return filepath.Join(s.storagePath, constants.RuntimeBuildPlanStore)
}

func (s *Store) buildScriptFile() string {
	return filepath.Join(s.storagePath, constants.BuildScriptStore)
}

// BuildEngine returns the runtime build engine value stored in the runtime directory
func (s *Store) BuildEngine() (types.BuildEngine, error) {
	storeFile := s.buildEngineFile()

	data, err := fileutils.ReadFile(storeFile)
	if err != nil {
		return types.UnknownEngine, errs.Wrap(err, "Could not read build engine cache store.")
	}

	return buildplanner.ParseBuildEngine(string(data)), nil
}

// StoreBuildEngine stores the build engine value in the runtime directory
func (s *Store) StoreBuildEngine(buildEngine types.BuildEngine) error {
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

	files, err := os.ReadDir(jsonDir)
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
func (s *Store) DeleteArtifactStore(id strfmt.UUID) error {
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

func (s *Store) UpdateEnviron(orderedArtifacts []strfmt.UUID) (*envdef.EnvironmentDefinition, error) {
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

func (s *Store) updateEnviron(orderedArtifacts []strfmt.UUID, artifacts StoredArtifactMap) (*envdef.EnvironmentDefinition, error) {
	if len(orderedArtifacts) == 0 {
		return nil, errs.New("Environment cannot be updated if no artifacts were installed")
	}

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

	if rtGlobal == nil {
		// Returning nil will end up causing a nil-pointer-exception panic in setup.Update().
		// There is additional logging of the buildplan there that may help diagnose why this is happening.
		logging.Error("There were artifacts returned, but none of them ended up being stored/installed.")
		logging.Error("Artifacts returned: %v", orderedArtifacts)
		logging.Error("Artifacts stored: %v", artifacts)
	}

	return rtGlobal, nil
}

// InstallPath returns the installation path of the runtime
func (s *Store) InstallPath() string {
	return s.installPath
}

var ErrNoBuildPlanFile = errs.New("no build plan file")

func (s *Store) BuildPlanRaw() ([]byte, error) {
	if !fileutils.FileExists(s.buildPlanFile()) {
		return nil, ErrNoBuildPlanFile
	}
	data, err := fileutils.ReadFile(s.buildPlanFile())
	if err != nil {
		return nil, errs.Wrap(err, "Could not read build plan file.")
	}

	return data, nil
}

func (s *Store) BuildPlan() (*buildplan.BuildPlan, error) {
	if !s.VersionMarkerIsValid() {
		return nil, locale.NewInputError("err_runtime_needs_refresh")
	}

	data, err := s.BuildPlanRaw()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get build plan file.")
	}

	return buildplan.Unmarshal(data)
}

func (s *Store) StoreBuildPlan(bp *buildplan.BuildPlan) error {
	data, err := bp.Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal buildPlan.")
	}
	err = fileutils.WriteFile(s.buildPlanFile(), data)
	if err != nil {
		return errs.Wrap(err, "Could not write recipe file.")
	}
	return nil
}

var ErrNoBuildScriptFile = errs.New("no buildscript file")

func (s *Store) BuildScript() (*buildscript.Script, error) {
	if !fileutils.FileExists(s.buildScriptFile()) {
		return nil, ErrNoBuildScriptFile
	}
	bytes, err := fileutils.ReadFile(s.buildScriptFile())
	if err != nil {
		return nil, errs.Wrap(err, "Could not read buildscript file")
	}
	return buildscript.New(bytes)
}

func (s *Store) StoreBuildScript(script *buildscript.Script) error {
	return fileutils.WriteFile(s.buildScriptFile(), []byte(script.String()))
}
