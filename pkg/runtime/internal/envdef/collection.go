package envdef

import (
	"encoding/json"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

// EnvironmentDefinitionFilename is the filename for runtime meta data bundled with artifacts, if they are built by the alternative builder
const EnvironmentDefinitionFilename = "runtime.json"

type raw struct {
	EnvDefs map[string]*EnvironmentDefinition `json:"Definitions"`
}

type Collection struct {
	raw  *raw // We use the raw struct so as to not directly expose the parsed JSON data to consumers
	path string
}

var ErrFileNotFound = errs.New("Environment definition file not found")

// NewCollection provides in-memory caching, and convenience layers for interacting with environment definitions
func NewCollection(path string) (*Collection, error) {
	c := &Collection{&raw{EnvDefs: map[string]*EnvironmentDefinition{}}, path}

	if !fileutils.TargetExists(path) {
		return c, ErrFileNotFound // Always return collection here, because this may not be a failure condition
	}

	b, err := fileutils.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to read environment definitions")
	}
	r := &raw{}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, errs.Wrap(err, "Failed to unmarshal environment definitions")
	}
	c.raw = r
	return c, nil
}

func (c *Collection) Save() error {
	b, err := json.Marshal(c.raw)
	if err != nil {
		return errs.Wrap(err, "Failed to marshal environment definitions")
	}
	if err := fileutils.WriteFile(c.path, b); err != nil {
		return errs.Wrap(err, "Failed to write environment definitions")
	}
	return nil
}

func (c *Collection) Load(path string) (*EnvironmentDefinition, error) {
	if envDef, ok := c.raw.EnvDefs[path]; ok {
		return envDef, nil
	}

	envDef, err := NewEnvironmentDefinition(filepath.Join(path, EnvironmentDefinitionFilename))
	if err != nil {
		return nil, errs.Wrap(err, "Failed to initialize environment definition")
	}
	c.raw.EnvDefs[path] = envDef
	return envDef, nil
}

func (c *Collection) Unload(path string) error {
	if _, ok := c.raw.EnvDefs[path]; !ok {
		return errs.New("Environment definition not found for path: %s", path)
	}
	delete(c.raw.EnvDefs, path)
	return nil
}

func (c *Collection) Environment() (map[string]string, error) {
	result := &EnvironmentDefinition{}
	var err error
	for _, envDef := range c.raw.EnvDefs {
		result, err = result.Merge(envDef)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to merge environment definitions")
		}
	}
	return result.GetEnv(false), nil
}
