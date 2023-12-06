package secrets

import (
	"fmt"

	"github.com/ActiveState/cli/internal/access"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/secrets"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
)

type listPrimeable interface {
	primer.Outputer
	primer.Projecter
	primer.Configurer
	primer.Auther
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
	cfg           keypairs.Configurable
	auth          *authentication.Auth
}

type secretData struct {
	Name        string `locale:"name,[HEADING]Name[/RESET]"`
	Scope       string `locale:"scope,[HEADING]Scope[/RESET]"`
	Description string `locale:"description,[HEADING]Description[/RESET]"`
	HasValue    string `locale:"hasvalue,[HEADING]Value[/RESET]"`
	Usage       string `locale:"usage,[HEADING]Usage[/RESET]"`
}

// NewList prepares a list execution context for use.
func NewList(client *secretsapi.Client, p listPrimeable) *List {
	return &List{
		secretsClient: client,
		out:           p.Output(),
		proj:          p.Project(),
		cfg:           p.Config(),
		auth:          p.Auth(),
	}
}

type listOutput struct {
	out  output.Outputer
	data []*secretData
}

func (o *listOutput) MarshalOutput(format output.Format) interface{} {
	return struct {
		Data []*secretData `opts:"verticalTable" locale:","`
	}{
		o.data,
	}
}

func (o *listOutput) MarshalStructured(format output.Format) interface{} {
	output := make([]*SecretExport, len(o.data))
	for i, d := range o.data {
		out := &SecretExport{
			Name:        d.Name,
			Scope:       d.Scope,
			Description: d.Description,
		}

		if d.HasValue == locale.T("secrets_row_value_set") {
			out.HasValue = true
		}

		output[i] = out
	}
	return output
}

// Run executes the list behavior.
func (l *List) Run(params ListRunParams) error {
	if l.proj == nil {
		return locale.NewInputError("err_no_project")
	}
	l.out.Notice(locale.Tr("operating_message", l.proj.NamespaceString(), l.proj.Dir()))

	if err := checkSecretsAccess(l.proj, l.auth); err != nil {
		return locale.WrapError(err, "secrets_err_check_access")
	}

	defs, err := definedSecrets(l.proj, l.secretsClient, l.cfg, l.auth, params.Filter)
	if err != nil {
		return locale.WrapError(err, "secrets_err_defined")
	}

	meta, err := defsToData(defs, l.cfg, l.proj, l.auth)
	if err != nil {
		return locale.WrapError(err, "secrets_err_values")
	}

	l.out.Print(&listOutput{l.out, meta})

	return nil
}

// checkSecretsAccess is reusable "runner-level" logic and provides a directly
// usable localized error.
func checkSecretsAccess(proj *project.Project, auth *authentication.Auth) error {
	if proj == nil {
		return locale.NewInputError("err_no_project")
	}
	allowed, err := access.Secrets(proj.Owner(), auth)
	if err != nil {
		return locale.WrapError(err, "secrets_err_access")
	}
	if !allowed {
		return locale.NewError("secrets_warning_no_access")
	}
	return nil
}

func definedSecrets(proj *project.Project, secCli *secretsapi.Client, cfg keypairs.Configurable, auth *authentication.Auth, filter string) ([]*secretsModels.SecretDefinition, error) {
	logging.Debug("listing variables for org=%s, project=%s", proj.Owner(), proj.Name())

	secretDefs, err := secrets.DefsByProject(secCli, proj.Owner(), proj.Name())
	if err != nil {
		return nil, err
	}

	if filter != "" {
		secretDefs = filterSecrets(proj, cfg, auth, secretDefs, filter)
	}

	return secretDefs, nil
}

func filterSecrets(proj *project.Project, cfg keypairs.Configurable, auth *authentication.Auth, secrectDefs []*secretsModels.SecretDefinition, filter string) []*secretsModels.SecretDefinition {
	secrectDefsFiltered := []*secretsModels.SecretDefinition{}

	oldExpander := project.RegisteredExpander("secrets")
	if oldExpander != nil {
		defer project.RegisterExpander("secrets", oldExpander)
	}
	expander := project.NewSecretExpander(secretsapi.Get(), proj, nil, cfg, auth)
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

func defsToData(defs []*secretsModels.SecretDefinition, cfg keypairs.Configurable, proj *project.Project, auth *authentication.Auth) ([]*secretData, error) {
	data := make([]*secretData, len(defs))
	expander := project.NewSecretExpander(secretsapi.Get(), proj, nil, cfg, auth)

	for i, def := range defs {
		if def.Name == nil || def.Scope == nil {
			multilog.Error("Could not get pointer for secret name and/or scope, definition ID: %s", def.DefID)
			continue
		}

		data[i] = &secretData{
			Name:        *def.Name,
			Scope:       *def.Scope,
			Description: def.Description,
			HasValue:    locale.T("secrets_row_value_unset"),
			Usage:       fmt.Sprintf("%s.%s", *def.Scope, *def.Name),
		}

		if data[i].Description == "" {
			data[i].Description = locale.T("secrets_description_unset")
		}

		secretValue, err := expander.FindSecret(*def.Name, *def.Scope == secretsModels.SecretDefinitionScopeUser)
		if err != nil {
			logging.Debug("Could not determine secret value, got error: %v", err)
			continue
		}

		if secretValue != nil {
			data[i].HasValue = locale.T("secrets_row_value_set")
		}
	}

	return data, nil
}
