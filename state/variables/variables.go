package variables

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
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
}

// NewCommand creates a new Keypair command.
func NewCommand(secretsClient *secretsapi.Client) *Command {
	c := Command{
		secretsClient: secretsClient,
		config: &commands.Command{
			Name:        "variables",
			Aliases:     []string{"vars"},
			Description: "variables_cmd_description",
		},
	}
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
	failure := listAllVariables(cmd.secretsClient)
	if failure != nil {
		failures.Handle(failure, locale.T("variables_err"))
	}
}

// listAllVariables prints a list of all of the variables defined for this project.
func listAllVariables(secretsClient *secretsapi.Client) *failures.Failure {
	prj := project.Get()
	logging.Debug("listing variables for org=%s, project=%s", prj.Owner(), prj.Name())

	vars, ff := makeVariables(prj.Variables())
	if ff != nil {
		return ff
	}

	hdrs, rows := variablesTable(vars)
	t := gotabulate.Create(rows)
	t.SetHeaders(hdrs)
	t.SetAlign("left")

	print.Line(t.Render("simple"))
	return nil
}

// variable represents data derived from a project.Variable value.
type variable struct {
	name      string
	desc      string
	setunset  string
	encrypted string
	shared    string
	store     string
}

func makeVariables(vars []*project.Variable) ([]variable, *failures.Failure) {
	var vs []variable

	for _, vx := range vars {
		valOrNil, ff := vx.ValueOrNil()
		if ff != nil {
			return nil, ff
		}

		issec := vx.IsSecret()
		isshr := vx.IsShared()
		shrWth := possibleString(vx.SharedWith())
		pldFrm := possibleString(vx.PulledFrom())

		v := variable{
			name:      vx.Name(),
			desc:      vx.Description(),
			setunset:  setOrUnset(valOrNil),
			encrypted: encVal(issec),
			shared:    sharedVal(issec, isshr, shrWth),
			store:     storeLocVal(issec, pldFrm),
		}
		vs = append(vs, v)
	}

	return vs, nil
}

func variablesTable(vars []variable) (hdrs []string, rows [][]string) {
	for _, v := range vars {
		row := []string{
			v.name,
			v.desc,
			v.setunset,
			v.encrypted,
			v.shared,
			v.store,
		}
		rows = append(rows, row)
	}

	hdrs = []string{
		locale.T("variables_col_name"),
		locale.T("variables_col_description"),
		locale.T("variables_col_setunset"),
		locale.T("variables_col_encrypted"),
		locale.T("variables_col_shared"),
		locale.T("variables_col_store"),
	}

	return hdrs, rows
}

func sharedVal(isSecret, isShared bool, sharedWith string) string {
	if isSecret && isShared {
		return sharedWith
	}
	return "-"
}

func storeLocVal(isSecret bool, pulledFrom string) string {
	if !isSecret {
		return "local"
	}
	return pulledFrom
}

func encVal(isSecret bool) string {
	if isSecret {
		return locale.T("confirmation")
	}
	return "-"
}

func setOrUnset(p *string) string {
	if p == nil {
		return locale.T("variables_value_unset")
	}
	return locale.T("variables_value_set")
}

func possibleString(i interface{}) string {
	if i == nil {
		return ""
	}

	switch v := i.(type) {
	case *projectfile.VariableShare:
		if v != nil {
			return string(*v)
		}
	case *projectfile.VariablePullFrom:
		if v != nil {
			return string(*v)
		}
	default:
	}

	return ""
}
