package alternative

import (
	"github.com/ActiveState/cli/pkg/project"
)

// var _ runtime.Runtimer = &Alternative{}

// Alternative is the specialization of a runtime for alternative builds
type Alternative struct{}

// New is the constructor function for alternative runtimes
func New(proj *project.Project) (*Alternative, error) {
	panic("implement me")
}

func (a *Alternative) Environ() (map[string]string, error) {
	panic("implement me")
}
