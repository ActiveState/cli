package projects

import (
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type EditParams struct {
	ProjectName string
	OwnerName   string
	Visibility  string
	Repository  string
}

type Edit struct {
	auth   *authentication.Auth
	out    output.Outputer
	prompt prompt.Prompter
	config configGetter
}

func NewEdit(prime primeable) *Edit {
	return &Edit{
		auth:   prime.Auth(),
		out:    prime.Output(),
		prompt: prime.Prompt(),
		config: prime.Config(),
	}
}

func (e *Edit) Run(params EditParams) error {
	return nil
}
