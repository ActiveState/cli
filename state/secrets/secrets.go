package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/access"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/project"
)

// Command represents the secrets command and its dependencies.
type Command struct {
	config        *commands.Command
	secretsClient *secretsapi.Client

	Args struct {
		Name            string
		Value           string
		ShareUserHandle string
	}

	Flags struct {
		Filter *string
		Output *string
	}
}

type SecretExport struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	HasValue    bool   `json:"has_value"`
	Value       string `json:"value,omitempty"`
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client, output *string) *Command {
	var flagFilter string

	c := Command{
		secretsClient: secretsClient,
		config: &commands.Command{
			Name:        "secrets",
			Aliases:     []string{"variables", "vars"},
			Description: "secrets_cmd_description",
			Flags: []*commands.Flag{
				{
					Name:        "filter-usedby",
					Description: "secrets_flag_filter",
					Type:        commands.TypeString,
					StringVar:   &flagFilter,
				},
			},
		},
	}

	c.Flags.Filter = &flagFilter
	c.Flags.Output = output
	c.config.Run = c.Execute
	c.config.PersistentPreRun = c.checkSecretsAccess

	c.config.Append(buildGetCommand(&c))
	c.config.Append(buildSetCommand(&c))
	c.config.Append(buildSyncCommand(&c))

	return &c
}

func (cmd *Command) checkSecretsAccess(_ *cobra.Command, _ []string) {
	allowed, fail := access.Secrets(project.Get().Owner())
	if fail != nil {
		failures.Handle(fail, locale.T("secrets_err_access"))
	}
	if !allowed {
		print.Warning(locale.T("secrets_warning_no_access"))
		cmd.config.Exiter(1)
	}
}

// Config returns the underlying commands.Command definition.
func (cmd *Command) Config() *commands.Command {
	return cmd.config
}

// Execute processes the secrets command.
func (cmd *Command) Execute(_ *cobra.Command, args []string) {
	if strings.HasPrefix(os.Args[1], "var") {
		print.Warning(locale.T("secrets_warn_deprecated_var"))
	}

	defs, fail := definedSecrets(cmd.secretsClient, *cmd.Flags.Filter)
	if fail != nil {
		failures.Handle(fail, locale.T("secrets_err_defined"))
		return
	}

	secretExports, fail := defsToSecrets(defs)
	if fail != nil {
		failures.Handle(fail, locale.T("secrets_err_values"))
		return
	}

	switch commands.Output(strings.ToLower(*cmd.Flags.Output)) {
	case commands.JSON, commands.EditorV0, commands.Editor:
		data, fail := secretsAsJSON(secretExports)
		if fail != nil {
			failures.Handle(fail, locale.T("secrets_err_output"))
			return
		}

		print.Line(string(data))
		return
	default:
		rows, fail := secretsToRows(secretExports)
		if fail != nil {
			failures.Handle(fail, locale.T("secrets_err_output"))
			return
		}

		t := gotabulate.Create(rows)
		t.SetHeaders([]string{locale.T("secrets_header_name"), locale.T("secrets_header_scope"), locale.T("secrets_header_value"), locale.T("secrets_header_description"), locale.T("secrets_header_usage")})
		t.SetHideLines([]string{"betweenLine", "top", "aboveTitle", "LineTop", "LineBottom", "bottomLine"}) // Don't print whitespace lines
		t.SetAlign("left")
		print.Line(t.Render("simple"))
	}
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
