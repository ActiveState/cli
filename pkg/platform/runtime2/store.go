package runtime

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
	"github.com/go-openapi/strfmt"
)

// Store manages the storing and loading of persistable information about the runtime
type Store struct {
	cachePath   string
	installPath string
}

// NewStore returns a new store instance
func NewStore(projectDir, cachePath string) (*Store, error) {
	projectDir = strings.TrimSuffix(projectDir, constants.ConfigFileName)
	resolvedProjectDir, err := fileutils.ResolveUniquePath(projectDir)
	if err != nil {
		return nil, locale.WrapError(err, "err_new_runtime_unique_path", "Failed to resolve a unique file path to the project dir.")
	}
	logging.Debug("In NewStore: resolved project dir is: %s", resolvedProjectDir)

	installPath := filepath.Join(cachePath, hash.ShortHash(resolvedProjectDir))
	return &Store{
		cachePath,
		installPath,
	}, nil
}

func (s *Store) markerFile() string {
	return filepath.Join(s.installPath, constants.RuntimeInstallationCompleteMarker)
}

func (s *Store) buildEngineFile() string {
	return filepath.Join(s.installPath, constants.RuntimeBuildEngineStore)
}

func (s *Store) recipeFile() string {
	return filepath.Join(s.installPath, constants.RuntimeRecipeStore)
}

// HasCompleteInstallation checks if stored runtime is complete and can be loaded
func (s *Store) HasCompleteInstallation(commitID strfmt.UUID) bool {
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

	logging.Debug("IsCachedRuntime for %s, %s==%s", marker, string(contents), commitID.String())
	return string(contents) == commitID.String()
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
func (s *Store) BuildEngine() (build.BuildEngine, error) {
	storeFile := s.buildEngineFile()

	data, err := fileutils.ReadFile(storeFile)
	if err != nil {
		return build.UnknownEngine, errs.Wrap(err, "Could not read build engine cache store.")
	}

	return build.ParseBuildEngine(string(data)), nil
}

// StoreBuildEngine stores the build engine value in the runtime directory
func (s *Store) StoreBuildEngine(buildEngine build.BuildEngine) error {
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

// InstallPath returns the installation path of the runtime
func (s *Store) InstallPath() string {
	return s.installPath
}
