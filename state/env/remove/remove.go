package remove

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/cmdlets/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"

	survey "gopkg.in/AlecAivazis/survey.v1"
)

// This is mostly a clone of the state/hooks/remove/remove.go file. Any bugfixes
// and changes in that file should be applied here and vice-versa.

// Args hold the arg values passed through the command line
var Args struct {
	Identifier string
}

// Used for testing
var testPromptResultOverride string

// Command remove, sub command of env
var Command = &commands.Command{
	Name:        "remove",
	Description: "env_remove_description",
	Run:         Execute,

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_env_remove_identifier",
			Description: "arg_env_remove_identifier_description",
			Variable:    &Args.Identifier,
		},
	},
}

// Execute the env remove command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute `env remove`")

	project := projectfile.Get()

	var removed *projectfile.Variable
	removed = removeByHash(Args.Identifier)

	if removed == nil {
		filters := []string{}
		if Args.Identifier != "" {
			filters = append(filters, Args.Identifier)
		}
		hashedVariables, err := variables.HashVariablesFiltered(project.Variables, filters)
		if err != nil {
			failures.Handle(err, locale.T("env_remove_cannot_remove"))
			return
		}

		numOfVariablesFound := len(hashedVariables)
		if numOfVariablesFound == 1 && Args.Identifier != "" {
			removed = removeByName(Args.Identifier)
		} else if numOfVariablesFound > 0 {
			removed = removeByPrompt(Args.Identifier)
		} else {
			failures.Handle(failures.FailUserInput.New("err_env_cannot_find"), "")
		}
	}

	if removed == nil {
		print.Warning(locale.T("env_remove_cannot_remove"))
	} else {
		hash, _ := removed.Hash()
		print.Info(locale.T("env_removed", map[string]interface{}{"Variable": removed.Name, "Hash": hash}))
	}
}

//  Cycle through the defined variables, hash then remove variable if matches, save, exit
func removeByHash(hashToRemove string) *projectfile.Variable {
	project := projectfile.Get()
	var removed *projectfile.Variable
	for i, variable := range project.Variables {
		hash, err := variable.Hash()
		if hashToRemove == hash {
			project.Variables = append(project.Variables[:i], project.Variables[i+1:]...)
			removed = &variable
			break
		} else if err != nil {
			logging.Warning("Failed to remove variable '%v': %v", hashToRemove, err)
			print.Warning(locale.T("env_remove_cannot_remove"))
		}
	}
	project.Save()
	return removed
}

func removeByName(name string) *projectfile.Variable {
	project := projectfile.Get()
	var removed *projectfile.Variable
	for i, variable := range project.Variables {
		if name == variable.Name {
			project.Variables = append(project.Variables[:i], project.Variables[i+1:]...)
			removed = &variable
			break
		}
	}
	project.Save()
	return removed
}

func removeByPrompt(identifier string) *projectfile.Variable {
	var removed *projectfile.Variable

	options, optionsMap, err := variables.PromptOptions(identifier)
	if err != nil {
		failures.Handle(err, locale.T("err_env_cannot_list"))
	}

	prompt := &survey.Select{
		Message: locale.T("prompt_env_choose_remove"),
		Options: options,
	}

	result := ""
	err = survey.AskOne(prompt, &result, nil)

	// For tests we want to override the result as we cannot process prompts from within a test
	if testPromptResultOverride != "" {
		result = testPromptResultOverride
	}

	if err != nil && testPromptResultOverride == "" {
		failures.Handle(err, locale.T("err_invalid_input"))
		return removed
	}

	hash, exists := optionsMap[result]
	if result == "" || !exists {
		print.Error(locale.T("err_env_cannot_find"))
		return removed
	}

	print.Line()
	return removeByHash(hash)
}
