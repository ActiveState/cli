package camel

import (
	"github.com/ActiveState/cli/pkg/platform/runtime2/common"
	"github.com/ActiveState/cli/pkg/project"
)

var _ common.Runtimer = &Camel{}

// Camel is the specialization of a runtime for Camel builds
type Camel struct{}

// New the constructor function for alternative runtimes
func New(proj *project.Project) (*Camel, error) {
	panic("implement me")
}

func (a *Camel) Environ() (map[string]string, error) {
	panic("implement me")
}
