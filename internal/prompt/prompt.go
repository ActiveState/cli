package prompt

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Prompter is the interface used to run our prompt from, useful for mocking in tests
type Prompter interface {
	Input(message, defaultResponse string, flags ...Flag) (string, *failures.Failure)
	InputAndValidate(message, defaultResponse string, validator func(val interface{}) error, flags ...Flag) (string, *failures.Failure)
	Select(message string, choices []string, defaultResponse string) (string, *failures.Failure)
	Confirm(message string, defaultChoice bool) (bool, *failures.Failure)
	InputPassword(message string) (string, *failures.Failure)
}

// FailPromptUnknownValidator handles unknown validator erros
var FailPromptUnknownValidator = failures.Type("prompt.unknownvalidator")

// Prompt is our main promptig struct
type Prompt struct{}

// New creates a new prompter
func New() Prompter {
	return &Prompt{}
}

// Flag represents flags for prompt functions to change their behavior on.
type Flag int

const (
	// InputRequired requires that the user provide input
	InputRequired Flag = iota
	// IsAlpha
	// IsNumber
	// etc.
)

// Input prompts the user for input.  The user can specify available validation flags to trigger validation of responses
func (p *Prompt) Input(message, defaultResponse string, flags ...Flag) (response string, fail *failures.Failure) {
	validators, fail := processFlags(flags)
	if fail != nil {
		return "", fail
	}

	response, fail = input(message, defaultResponse, wrapValidators(validators))
	return
}

// InputAndValidate prompts an input field and allows you to specfiy a custom validation function as well as the built in flags
func (p *Prompt) InputAndValidate(message, defaultResponse string, validator func(val interface{}) error, flags ...Flag) (response string, fail *failures.Failure) {
	validators, fail := processFlags(flags)
	if fail != nil {
		return "", fail
	}

	response, fail = input(message, defaultResponse, wrapValidators(append(validators, validator)))
	return
}

// Select prompts the user to select one entry from multiple choices
func (p *Prompt) Select(message string, choices []string, defaultChoice string) (response string, fail *failures.Failure) {
	err := survey.AskOne(&survey.Select{
		Message: formatMessage(message),
		Options: choices,
		Default: defaultChoice,
	}, &response, nil)
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return response, nil
}

// Confirm prompts user for yes or no response.
func (p *Prompt) Confirm(message string, defaultChoice bool) (bool, *failures.Failure) {
	var resp bool
	err := survey.AskOne(&survey.Confirm{
		Message: message,
		Default: defaultChoice,
	}, &resp, nil)
	if err != nil {
		return false, failures.FailUserInput.Wrap(err)
	}
	return resp, nil
}

// InputPassword prompts the user for input and obfuscates the text in stdout.
// Will fail if empty.
func (p *Prompt) InputPassword(message string) (response string, fail *failures.Failure) {
	err := survey.AskOne(&survey.Password{
		Message: formatMessage(message),
	}, &response, inputRequired) // passwords shouldn't be blank ever, right?
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return response, nil
}

// wrapValidators wraps a list of validators in a wrapper function that can be run by the survey package functions
func wrapValidators(validators []func(val interface{}) error) (validator func(val interface{}) error) {
	validator = func(val interface{}) error {
		for _, v := range validators {
			if error := v(val); error != nil {
				return error
			}
		}
		return nil
	}
	return
}

// Handle passing args from either Input... function to survey
func input(message, defaultResponse string, validator func(val interface{}) error) (response string, fail *failures.Failure) {
	err := survey.AskOne(&survey.Input{
		Message: formatMessage(message),
		Default: defaultResponse,
	}, &response, validator)
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return
}

// This function seems like overkill right now but the assumption is we'll have more than one built in validator
func processFlags(flags []Flag) (validators []func(val interface{}) error, fail *failures.Failure) {
	for flag := range flags {
		switch Flag(flag) {
		case InputRequired:
			validators = append(validators, inputRequired)
		default:
			fail = FailPromptUnknownValidator.New(locale.Tr("fail_prompt_bad_flag", string(flag)))
		}
	}
	return
}
