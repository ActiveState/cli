package envdef

import (
	"os"
	"path/filepath"
	"runtime"
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
	// Prevent concurrent reads and writes
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.raw.EnvDefs[path]; !ok {
		return errs.New("Environment definition not found for path: %s", path)
	}

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
	env := result.ExpandVariables(constants).GetEnv(inherit)
	promotePath(env)
	return env, nil
}

// promotPath is a temporary fix to ensure that the PATH is interpreted correctly on Windows
// Should be properly addressed by https://activestatef.atlassian.net/browse/DX-3030
func promotePath(env map[string]string) {
	if runtime.GOOS != "windows" {
		return
	}

	PATH, exists := env["PATH"]
	if !exists {
		return
	}

	// If Path exists, prepend PATH values to it
	Path, pathExists := env["Path"]
	if !pathExists {
		return
	}

	env["Path"] = PATH + string(os.PathListSeparator) + Path
	delete(env, "PATH")
}
