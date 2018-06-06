package inherit

import (
	"flag"
	"os"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"

	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Command holds the main definition for the env inherit command.
var Command = &commands.Command{
	Name:        "inherit",
	Description: "env_inherit_description",
	Run:         Execute,
}

// The list of environment variables to be inherited.
// This is used in order to prevent all environment variables from being
// inherited.
var recognizedVariables = []string{}

var testConfirm = true // for testing

// Execute inheriting environment variables.
// If any existing variables of the same name are going to be overwritten,
// prompt the user to confirm.
func Execute(cmd *cobra.Command, args []string) {
	project := projectfile.Get()
	for _, name := range recognizedVariables {
		found := false
		for i, variable := range project.Variables {
			if name != variable.Name {
				continue
			}
			var yes bool
			if flag.Lookup("test.v") == nil {
				prompt := &survey.Confirm{Message: locale.T("env_inherit_prompt_overwrite", map[string]string{
					"Name":     name,
					"OldValue": variable.Value,
					"NewValue": os.Getenv(name),
				})}
				err := survey.AskOne(prompt, &yes, nil)
				if err != nil {
					print.Error(locale.T("error_env_inherit_aborted"))
					return
				}
			} else {
				yes = testConfirm
			}
			if yes {
				project.Variables[i].Value = os.Getenv(name)
			}
			found = true
			break
		}
		if !found {
			variable := projectfile.Variable{Name: name, Value: os.Getenv(name)}
			project.Variables = append(project.Variables, variable)
		}
	}
	project.Save()
}
