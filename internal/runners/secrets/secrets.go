package secrets

import (
	"fmt"

	"github.com/ActiveState/cli/internal/access"
	"github.com/ActiveState/cli/internal/keypairs"
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
	primer.Projecter
	primer.Configurer
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
}

type secretData struct {
	Name        string `locale:"name,[HEADING]Name[/RESET]"`
	Scope       string `locale:"scope,[HEADING]Scope[/RESET]"`
	Description string `locale:"description,[HEADING]Description[/RESET]"`
	HasValue    string `locale:"hasvalue,[HEADING]Value[/RESET]"`
	Usage       string `locale:"usage,[HEADING]Usage[/RESET]"`
}

type listOutput struct {
	out  output.Outputer
	data []*secretData
}

// NewList prepares a list execution context for use.
func NewList(client *secretsapi.Client, p listPrimeable) *List {
	return &List{
		secretsClient: client,
		out:           p.Output(),
		proj:          p.Project(),
		cfg:           p.Config(),
	}
}

// Run executes the list behavior.
func (l *List) Run(params ListRunParams) error {
	if l.proj == nil {
		return locale.NewInputError("err_no_project")
	}
	if err := checkSecretsAccess(l.proj); err != nil {
		return locale.WrapError(err, "secrets_err_check_access")
	}

	defs, err := definedSecrets(l.proj, l.secretsClient, l.cfg, params.Filter)
	if err != nil {
		return locale.WrapError(err, "secrets_err_defined")
	}

	meta, err := defsToData(defs, l.cfg, l.proj)
	if err != nil {
		return locale.WrapError(err, "secrets_err_values")
	}

	data := &listOutput{l.out, meta}
	l.out.Print(data)

	return nil
}

func (l *listOutput) MarshalOutput(format output.Format) interface{} {
	switch format {
	case output.EditorV0FormatName:
		var output []*SecretExport
		for _, d := range l.data {
			out := &SecretExport{
				Name:        d.Name,
				Scope:       d.Scope,
				Description: d.Description,
			}

			if d.HasValue == locale.T("secrets_row_value_set") {
				out.HasValue = true
			}

			output = append(output, out)
		}
		l.out.Print(output)
	default:
		l.out.Print(struct {
			Data []*secretData `opts:"verticalTable" locale:","`
		}{
			l.data,
		})
	}

	return output.Suppress
}

// checkSecretsAccess is reusable "runner-level" logic and provides a directly
// usable localized error.
func checkSecretsAccess(proj *project.Project) error {
	allowed, err := access.Secrets(proj.Owner())
	if err != nil {
		return locale.WrapError(err, "secrets_err_access")
	}
	if !allowed {
		return locale.NewError("secrets_warning_no_access")
	}
	return nil
}

func definedSecrets(proj *project.Project, secCli *secretsapi.Client, cfg keypairs.Configurable, filter string) ([]*secretsModels.SecretDefinition, error) {
	logging.Debug("listing variables for org=%s, project=%s", proj.Owner(), proj.Name())

	secretDefs, err := secrets.DefsByProject(secCli, proj.Owner(), proj.Name())
	if err != nil {
		return nil, err
	}

	if filter != "" {
		secretDefs = filterSecrets(proj, cfg, secretDefs, filter)
	}

	return secretDefs, nil
}

func filterSecrets(proj *project.Project, cfg keypairs.Configurable, secrectDefs []*secretsModels.SecretDefinition, filter string) []*secretsModels.SecretDefinition {
	secrectDefsFiltered := []*secretsModels.SecretDefinition{}

	oldExpander := project.RegisteredExpander("secrets")
	if oldExpander != nil {
		defer project.RegisterExpander("secrets", oldExpander)
	}
	expander := project.NewSecretExpander(secretsapi.Get(), proj, nil, cfg)
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

func defsToData(defs []*secretsModels.SecretDefinition, cfg keypairs.Configurable, proj *project.Project) ([]*secretData, error) {
	data := make([]*secretData, len(defs))
	expander := project.NewSecretExpander(secretsapi.Get(), proj, nil, cfg)

	for i, def := range defs {
		if def.Name == nil || def.Scope == nil {
			logging.Error("Could not get pointer for secret name and/or scope, definition ID: %d", def.DefID)
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
