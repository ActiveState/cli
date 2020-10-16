package secrets

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

type getPrimeable interface {
	primer.Outputer
	primer.Projecter
}

type GetRunParams struct {
	Name string
}

type Get struct {
	proj *project.Project
	out  output.Outputer
}

func NewGet(p getPrimeable) *Get {
	return &Get{
		out:  p.Output(),
		proj: p.Project(),
	}
}

func (g *Get) Run(params GetRunParams) error {
	if err := checkSecretsAccess(g.proj); err != nil {
		return err
	}

	secret, valuePtr, fail := getSecretWithValue(params.Name)
	if fail != nil {
		return fail.WithDescription(locale.T("secrets_err"))
	}

	var value string
	if valuePtr == nil {
		value = ""
	} else {
		value = *valuePtr
	}

	switch g.out.Type() {
	case output.JSONFormatName, output.EditorV0FormatName, output.EditorFormatName:
		fail := printJSON(&SecretExport{secret.Name(), secret.Scope(), secret.Description(), valuePtr != nil, value})
		if fail != nil {
			return fail.WithDescription(locale.T("secrets_err"))
		}
	default:
		if valuePtr == nil {
			l10nKey := "secrets_err_project_not_defined"
			if secret.IsUser() {
				l10nKey = "secrets_err_user_not_defined"
			}
			return locale.NewError(l10nKey, params.Name)
		}
		fmt.Fprint(os.Stdout, *valuePtr)
	}

	return nil
}

func printJSON(secretJSON *SecretExport) *failures.Failure {
	var data []byte

	data, err := json.Marshal(secretJSON)
	if err != nil {
		return failures.FailMarshal.Wrap(err)
	}

	fmt.Fprint(os.Stdout, string(data))
	return nil
}
