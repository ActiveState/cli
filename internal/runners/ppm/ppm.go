package ppm

import (
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/project"
)

// Ppm is the runner struct for ppm functionality
type Ppm struct {
	prompt  prompt.Prompter
	out     output.Outputer
	project *project.Project
}

type primeable interface {
	primer.Prompter
	primer.Outputer
	primer.Projecter
}

// New creates a new ppm runner
func New(prime primeable) *Ppm {
	return &Ppm{
		prime.Prompt(),
		prime.Output(),
		prime.Project(),
	}
}
