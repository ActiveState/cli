package camel

import (
	"github.com/ActiveState/cli/pkg/project"
)

// var _ runtime.EnvProvider = &Camel{}

// Camel is the specialization of a runtime for Camel builds
type Camel struct{}

// New is the constructor function for Camel runtimes
func New(proj *project.Project) (*Camel, error) {
	panic("implement me")
}

// Environ returns the environment variables needed to use this runtime
func (a *Camel) Environ() (map[string]string, error) {
	panic("implement me")
}
