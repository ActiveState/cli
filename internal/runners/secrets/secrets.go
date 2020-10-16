package secrets

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bndr/gotabulate"

	"github.com/ActiveState/cli/internal/access"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/secrets"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/project"
)

type listPrimeable interface {
	primer.Outputer
}

type ListRunParams struct {
	Filter string
}

type List struct {
	secretsClient *secretsapi.Client
	out           output.Outputer
}

func NewList(client *secretsapi.Client, p listPrimeable) *List {
	return &List{
		secretsClient: client,
		out:           p.Output(),
	}
}

func (l *List) Run(params ListRunParams) error {
	//c.config.PersistentPreRun = c.checkSecretsAccess
	defs, fail := definedSecrets(l.secretsClient, params.Filter)
	if fail != nil {
		return fail.WithDescription(locale.T("secrets_err_defined"))
	}

	secretExports, fail := defsToSecrets(defs)
	if fail != nil {
		return fail.WithDescription(locale.T("secrets_err_values"))
	}

	switch l.out.Type() {
	case output.JSONFormatName, output.EditorV0FormatName, output.EditorFormatName:
		data, fail := secretsAsJSON(secretExports)
		if fail != nil {
			return fail.WithDescription(locale.T("secrets_err_output"))
		}

		fmt.Fprint(os.Stdout, string(data))
		return nil
	default:
		rows, fail := secretsToRows(secretExports)
		if fail != nil {
			return fail.WithDescription(locale.T("secrets_err_output"))
		}

		t := gotabulate.Create(rows)
		t.SetHeaders([]string{locale.T("secrets_header_name"), locale.T("secrets_header_scope"), locale.T("secrets_header_value"), locale.T("secrets_header_description"), locale.T("secrets_header_usage")})
		t.SetHideLines([]string{"betweenLine", "top", "aboveTitle", "LineTop", "LineBottom", "bottomLine"}) // Don't print whitespace lines
		t.SetAlign("left")
		fmt.Fprint(os.Stdout, t.Render("simple"))
		return nil
	}
}

func CheckSecretsAccess() error {
	allowed, fail := access.Secrets(project.Get().Owner())
	if fail != nil {
		return fail.WithDescription(locale.T("secrets_err_access"))
	}
	if !allowed {
		return locale.NewError("secrets_warning_no_access")
	}
	return nil
}

func definedSecrets(secCli *secretsapi.Client, filter string) ([]*secretsModels.SecretDefinition, *failures.Failure) {
	prj := project.Get()
	logging.Debug("listing variables for org=%s, project=%s", prj.Owner(), prj.Name())

	secretDefs, fail := secrets.DefsByProject(secCli, prj.Owner(), prj.Name())
	if fail != nil {
		return nil, fail
	}

	if filter != "" {
		secretDefs = filterSecrets(secretDefs, filter)
	}

	return secretDefs, nil
}

func filterSecrets(secrectDefs []*secretsModels.SecretDefinition, filter string) []*secretsModels.SecretDefinition {
	prj := project.Get()
	secrectDefsFiltered := []*secretsModels.SecretDefinition{}

	oldExpander := project.RegisteredExpander("secrets")
	if oldExpander != nil {
		defer project.RegisterExpander("secrets", oldExpander)
	}

	expander := project.NewSecretExpander(secretsapi.Get(), prj)
	project.RegisterExpander("secrets", expander.Expand)
	project.ExpandFromProject(fmt.Sprintf("$%s", filter), prj)
	accessedSecrets := expander.SecretsAccessed()
	if accessedSecrets == nil {
		return secrectDefsFiltered
	}

	for _, secretDef := range secrectDefs {
		isUser := *secretDef.Scope == secretsModels.SecretDefinitionScopeUser
		for _, accessedSecret := range accessedSecrets {
			if accessedSecret.Name == *secretDef.Name && accessedSecret.IsUser == isUser {
				secrectDefsFiltered = append(secrectDefsFiltered, secretDef)
			}
		}
	}

	return secrectDefsFiltered
}

func defsToSecrets(defs []*secretsModels.SecretDefinition) ([]*SecretExport, *failures.Failure) {
	secretsExport := make([]*SecretExport, len(defs))
	expander := project.NewSecretExpander(secretsapi.Get(), project.Get())

	for i, def := range defs {
		if def.Name == nil || def.Scope == nil {
			logging.Error("Could not get pointer for secret name and/or scope, definition ID: %d", def.DefID)
			continue
		}

		secretValue, fail := expander.FindSecret(*def.Name, *def.Scope == secretsModels.SecretDefinitionScopeUser)
		if fail != nil {
			return secretsExport, fail
		}

		secretsExport[i] = &SecretExport{
			Name:        *def.Name,
			Scope:       *def.Scope,
			Description: def.Description,
			HasValue:    secretValue != nil,
		}
	}

	return secretsExport, nil
}

func secretsAsJSON(secretExports []*SecretExport) ([]byte, *failures.Failure) {
	bs, err := json.Marshal(secretExports)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return bs, nil
}

// secretsToRows returns the rows used in our output table
func secretsToRows(secretExports []*SecretExport) ([][]interface{}, *failures.Failure) {
	rows := [][]interface{}{}
	for _, secret := range secretExports {
		description := "-"
		if secret.Description != "" {
			description = secret.Description
		}
		hasValue := locale.T("secrets_row_value_set")
		if !secret.HasValue {
			hasValue = locale.T("secrets_row_value_unset")
		}
		rows = append(rows, []interface{}{secret.Name, secret.Scope, hasValue, description, fmt.Sprintf("%s.%s", secret.Scope, secret.Name)})
	}
	return rows, nil
}

func ptrToString(s *string, fieldName string) (string, *failures.Failure) {
	if s == nil {
		return "", failures.FailVerify.New("secrets_err_missing_field", fieldName)
	}
	return *s, nil
}

type SecretExport struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	HasValue    bool   `json:"has_value"`
	Value       string `json:"value,omitempty"`
}
