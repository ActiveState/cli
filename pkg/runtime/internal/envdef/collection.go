package envdef

import "github.com/ActiveState/cli/internal/errs"

type Collection struct {
	envDefs map[string]*EnvironmentDefinition
}

// NewCollection provides in-memory caching, and convenience layers for interacting with environment definitions
func NewCollection() *Collection {
	return &Collection{
		envDefs: make(map[string]*EnvironmentDefinition),
	}
}

func (c *Collection) Get(path string) (*EnvironmentDefinition, error) {
	if envDef, ok := c.envDefs[path]; ok {
		return envDef, nil
	}

	envDef, err := NewEnvironmentDefinition(path)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to initialize environment definition")
	}
	c.envDefs[path] = envDef
	return envDef, nil
}

func (c *Collection) Environment() (map[string]string, error) {
	result := &EnvironmentDefinition{}
	var err error
	for _, envDef := range c.envDefs {
		result, err = result.Merge(envDef)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to merge environment definitions")
		}
	}
	return result.GetEnv(false), nil
}
