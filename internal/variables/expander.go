package variables

import (
	"github.com/ActiveState/cli/internal/secrets"

	"github.com/ActiveState/cli/internal/failures"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var (
	// FailVarNotFound is used when no handler func is found for an Expander.
	FailVarNotFound = failures.Type("variables.fail.notfound", FailExpandVariable)
)

type VarExpander struct {
	secretsClient   *secretsapi.Client
	secretsExpander secrets.ExpanderFunc
}

func (e *VarExpander) Expand(name string, projectFile *projectfile.Project) (string, *failures.Failure) {
	var variable *projectfile.Variable
	for _, varcheck := range projectFile.Variables {
		if varcheck.Name == name {
			variable = varcheck
			break
		}
	}

	if variable == nil {
		return "", FailVarNotFound.New("variables_expand_err_spec_undefined", name)
	}

	if variable.Value.StaticValue != nil {
		return *variable.Value.StaticValue, nil
	}

	return e.secretsExpander(variable, projectFile)
}

// NewExpander creates an ExpanderFunc which can retrieve and decrypt stored user secrets.
func NewVarExpanderFunc(secretsClient *secretsapi.Client) ExpanderFunc {
	secretsExpander := secrets.NewExpander(secretsClient)
	expander := &VarExpander{secretsClient, secretsExpander.Expand}
	return expander.Expand
}

// NewPromptingExpander creates an ExpanderFunc which can retrieve and decrypt stored user secrets. Additionally,
// it will prompt the user to provide a value for a secret -- in the event none is found -- and save the new
// value with the secrets service.
func NewVarPromptingExpanderFunc(secretsClient *secretsapi.Client) ExpanderFunc {
	secretsExpander := secrets.NewExpander(secretsClient)
	expander := &VarExpander{secretsClient, secretsExpander.ExpandWithPrompt}
	return expander.Expand
}
