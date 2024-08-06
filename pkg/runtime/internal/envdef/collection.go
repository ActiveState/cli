package envdef

import (
	"path/filepath"
	"sync"

	"github.com/ActiveState/cli/internal/errs"
)

// EnvironmentDefinitionFilename is the filename for runtime meta data bundled with artifacts, if they are built by the alternative builder
const EnvironmentDefinitionFilename = "runtime.json"

type raw struct {
	EnvDefs map[string]*EnvironmentDefinition `json:"Definitions"`
}

type Collection struct {
	raw   *raw // We use the raw struct so as to not directly expose the parsed JSON data to consumers
	mutex *sync.Mutex
}

var ErrFileNotFound = errs.New("Environment definition file not found")

func New() *Collection {
	return &Collection{&raw{EnvDefs: map[string]*EnvironmentDefinition{}}, &sync.Mutex{}}
}

func (c *Collection) Load(path string) (*EnvironmentDefinition, error) {
	if envDef, ok := c.raw.EnvDefs[path]; ok {
		return envDef, nil
	}

	envDef, err := NewEnvironmentDefinition(filepath.Join(path, EnvironmentDefinitionFilename))
	if err != nil {
		return nil, errs.Wrap(err, "Failed to initialize environment definition")
	}

	// Prevent concurrent writes
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.raw.EnvDefs[path] = envDef
	return envDef, nil
}

func (c *Collection) Unload(path string) error {
	if _, ok := c.raw.EnvDefs[path]; !ok {
		return errs.New("Environment definition not found for path: %s", path)
	}

	// Prevent concurrent writes
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.raw.EnvDefs, path)

	return nil
}

func (c *Collection) Environment(installPath string, inherit bool) (map[string]string, error) {
	result := &EnvironmentDefinition{}
	var err error
	for _, envDef := range c.raw.EnvDefs {
		result, err = result.Merge(envDef)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to merge environment definitions")
		}
	}
	constants := NewConstants(installPath)
	return result.ExpandVariables(constants).GetEnv(inherit), nil
}
