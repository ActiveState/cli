package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"

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
		JSON *bool
	}
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client) *Command {
	var flagJSON bool

	c := Command{
		secretsClient: secretsClient,
		config: &commands.Command{
			Name:        "secrets",
			Aliases:     []string{"variables", "vars"},
			Description: "secrets_cmd_description",
			Flags: []*commands.Flag{
				{
					Name:        "json",
					Description: "secrets_flag_json",
					Type:        commands.TypeBool,
					BoolVar:     &flagJSON,
				},
			},
		},
	}

	c.Flags.JSON = &flagJSON
	c.config.Run = c.Execute

	c.config.Append(buildGetCommand(&c))
	c.config.Append(buildSetCommand(&c))
	c.config.Append(buildSyncCommand(&c))

	return &c
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

	defs, fail := definedSecrets(cmd.secretsClient)
	if fail != nil {
		failures.Handle(fail, locale.T("secrets_err_defined"))
		return
	}

	if *cmd.Flags.JSON {
		data, fail := secretsAsJSON(defs)
		if fail != nil {
			failures.Handle(fail, locale.T("secrets_err_output"))
			return
		}

		print.Line(string(data))
		return
	}

	rows, fail := secretsToRows(defs)
	if fail != nil {
		failures.Handle(fail, locale.T("secrets_err_output"))
		return
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("secrets_header_name"), locale.T("secrets_header_scope"), locale.T("secrets_header_description"), locale.T("secrets_header_usage")})
	t.SetHideLines([]string{"betweenLine", "top", "aboveTitle", "LineTop", "LineBottom", "bottomLine"}) // Don't print whitespace lines
	t.SetAlign("left")
	print.Line(t.Render("simple"))
}

func definedSecrets(secCli *secretsapi.Client) ([]*secretsModels.SecretDefinition, *failures.Failure) {
	prj := project.Get()
	logging.Debug("listing variables for org=%s, project=%s", prj.Owner(), prj.Name())

	return secrets.DefsByProject(secCli, prj.Owner(), prj.Name())
}

func secretsAsJSON(defs []*secretsModels.SecretDefinition) ([]byte, *failures.Failure) {
	type secretDefinition struct {
		Name        string `json:"name,omitempty"`
		Scope       string `json:"scope,omitempty"`
		Description string `json:"desc,omitempty"`
		Value       string `json:"value,omitempty"`
	}

	ds := make([]secretDefinition, len(defs))

	for i, def := range defs {
		name, fail := ptrToString(def.Name, "name")
		if fail != nil {
			return nil, fail
		}
		scope, fail := ptrToString(def.Scope, "scope")
		if fail != nil {
			return nil, fail
		}

		secretKey := scope + "." + name

		_, value, fail := getSecretWithValue(secretKey)
		if fail != nil {
			return nil, fail
		}

		ds[i] = secretDefinition{
			Name:        name,
			Scope:       scope,
			Description: def.Description,
			Value:       ptrToStringWithDefault(value, ""),
		}
	}

	bs, err := json.Marshal(ds)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return bs, nil
}

// secretsToRows returns the rows used in our output table
func secretsToRows(defs []*secretsModels.SecretDefinition) ([][]interface{}, *failures.Failure) {
	rows := [][]interface{}{}
	for _, def := range defs {
		description := "-"
		if def.Description != "" {
			description = def.Description
		}
		rows = append(rows, []interface{}{*def.Name, *def.Scope, description, fmt.Sprintf("%s.%s", *def.Scope, *def.Name)})
	}
	return rows, nil
}

func ptrToString(s *string, fieldName string) (string, *failures.Failure) {
	if s == nil {
		return "", failures.FailVerify.New("secrets_err_missing_field", fieldName)
	}
	return *s, nil
}

func ptrToStringWithDefault(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}
