package secrets

import (
	"encoding/json"
	"fmt"

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
	"github.com/bndr/gotabulate"
)

type listPrimeable interface {
	primer.Outputer
	primer.Projecter
}

// ListRunParams tracks the info required for running List.
type ListRunParams struct {
	Filter string
}

// List manages the listing execution context.
type List struct {
	secretsClient *secretsapi.Client
	out           output.Outputer
	proj          *project.Project
}

// NewList prepares a list execution context for use.
func NewList(client *secretsapi.Client, p listPrimeable) *List {
	return &List{
		secretsClient: client,
		out:           p.Output(),
		proj:          p.Project(),
	}
}

// Run executes the list behavior.
func (l *List) Run(params ListRunParams) error {
	if err := checkSecretsAccess(l.proj); err != nil {
		return err
	}

	defs, fail := definedSecrets(l.proj, l.secretsClient, params.Filter)
	if fail != nil {
		return locale.WrapError(fail, "secrets_err_defined")
	}
	exports, fail := defsToSecrets(defs)
	if fail != nil {
		return locale.WrapError(fail, "secrets_err_values")
	}

	l.out.Print(secretExports(exports))

	return nil
}

// checkSecretsAccess is reusable "runner-level" logic and provides a directly
// usable error.
func checkSecretsAccess(proj *project.Project) error {
	allowed, fail := access.Secrets(proj.Owner())
	if fail != nil {
		return locale.WrapError(fail, "secrets_err_access")
	}
	if !allowed {
		return locale.NewError("secrets_warning_no_access")
	}
	return nil
}

func definedSecrets(proj *project.Project, secCli *secretsapi.Client, filter string) ([]*secretsModels.SecretDefinition, *failures.Failure) {
	logging.Debug("listing variables for org=%s, project=%s", proj.Owner(), proj.Name())

	secretDefs, fail := secrets.DefsByProject(secCli, proj.Owner(), proj.Name())
	if fail != nil {
		return nil, fail
	}

	if filter != "" {
		secretDefs = filterSecrets(proj, secretDefs, filter)
	}

	return secretDefs, nil
}

func filterSecrets(proj *project.Project, secrectDefs []*secretsModels.SecretDefinition, filter string) []*secretsModels.SecretDefinition {
	secrectDefsFiltered := []*secretsModels.SecretDefinition{}

	oldExpander := project.RegisteredExpander("secrets")
	if oldExpander != nil {
		defer project.RegisterExpander("secrets", oldExpander)
	}
	expander := project.NewSecretExpander(secretsapi.Get(), proj)
	project.RegisterExpander("secrets", expander.Expand)
	project.ExpandFromProject(fmt.Sprintf("$%s", filter), proj)
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

type secretExports []*SecretExport

func (es secretExports) MarshalOutput(format output.Format) interface{} {
	switch format {
	case output.JSONFormatName, output.EditorV0FormatName, output.EditorFormatName:
		return es

	default:
		rows, fail := secretsToRows(es)
		if fail != nil {
			return fail.WithDescription(locale.T("secrets_err_output"))
		}

		t := gotabulate.Create(rows)
		t.SetHeaders([]string{locale.T("secrets_header_name"), locale.T("secrets_header_scope"), locale.T("secrets_header_value"), locale.T("secrets_header_description"), locale.T("secrets_header_usage")})
		t.SetHideLines([]string{"betweenLine", "top", "aboveTitle", "LineTop", "LineBottom", "bottomLine"}) // Don't print whitespace lines
		t.SetAlign("left")

		return t.Render("simple")
	}
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

// SecretExport defines important information about a secret that should be
// displayed.
type SecretExport struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	HasValue    bool   `json:"has_value"`
	Value       string `json:"value,omitempty"`
}
