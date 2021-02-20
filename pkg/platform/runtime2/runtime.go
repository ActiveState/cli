package runtime

import (
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
	"github.com/ActiveState/cli/pkg/project"
)

type EnvProvider interface {
	Environ() (map[string]string, error)
}

type Runtime struct {
	proj *project.Project
	ep   EnvProvider
}

// new is the constructor function for alternative runtimes
func new(proj *project.Project, ep EnvProvider) (*Runtime, error) {
	r := Runtime{
		proj: proj,
		ep:   ep,
	}
	return &r, nil
}

func (r *Runtime) Environ() (map[string]string, error) {
	return r.ep.Environ()
}

func (r *Runtime) Artifacts() (map[build.ArtifactID]build.Artifact, error) {
	// read in recipe stored on disk and transform into artifact
	panic("implement me")
}
