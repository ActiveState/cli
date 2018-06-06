package add

import (
	"fmt"
	"os"
	"regexp"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/cmdlets/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
)

// This is mostly a clone of the state/hooks/add/add.go file. Any bugfixes and
// changes in that file should be applied here and vice-versa.

// Args hold the arg values passed through the command line
var Args struct {
	Name  string
	Value string
}

// Command Add
var Command = &commands.Command{
	Name:        "add",
	Description: "env_add_description",
	Run:         Execute,

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_env_add_variable",
			Description: "arg_env_add_variable_description",
			Variable:    &Args.Name,
			Required:    true,
			Validator: func(arg *commands.Argument, value string) error {
				regex := regexp.MustCompile("^\\w+$")
				if !regex.MatchString(value) {
					return failures.FailUserInput.New(locale.T("err_env_add_invalid_variable", map[string]interface{}{"Name": value}))
				}
				return nil
			},
		},
		&commands.Argument{
			Name:        "arg_env_add_value",
			Description: "env_hook_add_value_description",
			Variable:    &Args.Value,
			Required:    false,
		},
	},
}

// Execute the env add command
func Execute(cmd *cobra.Command, args []string) {
	// Add variable to activestate.yaml for the active project
	project := projectfile.Get()

	value := Args.Value
	if value == "" {
		value = os.Getenv(Args.Name)
	}
	newVariable := projectfile.Variable{Name: Args.Name, Value: value}

	exists, err := variables.VariableExists(newVariable, project)
	if err != nil {
		failures.Handle(err, locale.T("env_add_cannot_add_variable", Args))
		return
	}
	if exists {
		fmt.Printf(locale.T("env_add_cannot_add_existing_variable"))
		return
	}
	project.Variables = append(project.Variables, newVariable)
	project.Save()
	logging.Debug("Execute `hook add`")
}
