package prompt

import (
	"github.com/ActiveState/cli/internal/failures"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Prompter is the interface used to run our prompt from, useful for mocking in tests
type Prompter interface {
	Input(message, response string, validator func(val interface{}) error) (string, *failures.Failure)
	Select(message string, choices []string, response string) (string, *failures.Failure)
	Confirm(message string, defaultChoice bool) (bool, *failures.Failure)
	InputPassword(message string) (string, *failures.Failure)
}

// Prompt is our main promptig struct
type Prompt struct{}

// New creates a new prompter
func New() Prompter {
	return &Prompt{}
}

// Input prompts the user for input
func (p *Prompt) Input(message, defaultResponse string, validator func(val interface{}) error) (response string, fail *failures.Failure) {
	err := survey.AskOne(&survey.Input{
		Message: formatMessage(message),
		Default: defaultResponse,
	}, &response, validator)
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return response, nil
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
