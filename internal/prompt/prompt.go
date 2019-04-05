package prompt

import (
	"fmt"

	"github.com/ActiveState/cli/internal/failures"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Prompter is the interface used to run our prompt from, useful for mocking in tests
type Prompter interface {
	Input(message, response string, flags ...Flag) (string, *failures.Failure)
	InputAndValidate(message, response string, validator func(val interface{}) error) (string, *failures.Failure)
	Select(message string, choices []string, response string) (string, *failures.Failure)
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
	// NoValidation don't validate the input
	NoValidation Flag = iota
	// InputRequired requires that the user provide input
	InputRequired
)

// Input prompts the user for input
func (p *Prompt) Input(message, defaultResponse string, flags ...Flag) (response string, fail *failures.Failure) {
	validators, fail := processFlags(flags)
	if fail != nil {
		return "", fail
	}
	validator := func(val interface{}) error {
		for _, v := range validators {
			if error := v(val); error != nil {
				return error
			}
		}
		return nil
	}
	response, fail = input(message, defaultResponse, validator)
	return
}

// InputAndValidate prompts an input field and allows you to specfiy a custom validation function
func (p *Prompt) InputAndValidate(message, defaultResponse string, validator func(val interface{}) error) (response string, fail *failures.Failure) {
	response, fail = input(message, defaultResponse, validator)
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
	}, &response, ValidateRequired) // passwords shouldn't be blank ever, right?
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return response, nil
}

func input(message, defaultResponse string, validator func(val interface{}) error) (response string, fail *failures.Failure) {
	err := survey.AskOne(&survey.Input{
		Message: formatMessage(message),
		Default: defaultResponse,
	}, &response, validator)
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return response, nil
}

func processFlags(flags []Flag) (validators []func(val interface{}) error, fail *failures.Failure) {
	for flag := range flags {
		switch Flag(flag) {
		case InputRequired:
			validators = append(validators, ValidateRequired)
		case NoValidation:
			validators = append(validators, NoValidate)
		default:
			fail = FailPromptUnknownValidator.New(fmt.Sprintf("Unknown Prompt flag: %d", flag))
		}
	}
	return
}
