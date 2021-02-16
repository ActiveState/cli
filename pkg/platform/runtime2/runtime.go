package runtime

import (
	"github.com/ActiveState/cli/pkg/project"
)

type EnvProvider interface {
	Environ() (map[string]string, error)
}

type Runtime struct {
	proj *project.Project
	ep   EnvProvider
}

// New is the constructor function for alternative runtimes
func New(proj *project.Project, ep EnvProvider) (*Runtime, error) {
	r := Runtime{
		proj: proj,
		ep:   ep,
	}
	return &r, nil
}

func (r *Runtime) Environ() (map[string]string, error) {
	return r.ep.Environ()
}
