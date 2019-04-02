package prompt

import (
	"github.com/ActiveState/cli/internal/failures"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Prompter is the interface used to run our prompt from, useful for mocking in tests
type Prompter interface {
	Input(message string, response string) (string, *failures.Failure)
	Select(message string, choices []string, response string) (string, *failures.Failure)
	Confirm(message string) (bool, *failures.Failure)
}

// Prompt is our main promptig struct
type Prompt struct{}

var prompter Prompter

// New creates a new prompter
func New() Prompter {
	return &Prompt{}
}

func init() {
	prompter = New()
}

// Input prompts the user for input
func (p *Prompt) Input(message string, response string) (string, *failures.Failure) {
	err := survey.AskOne(&survey.Input{
		Message: formatMessage(message),
		Default: response,
	}, &response, nil)
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return response, nil
}

// Select prompts the user to select one entry from multiple choices
func (p *Prompt) Select(message string, choices []string, response string) (string, *failures.Failure) {
	err := survey.AskOne(&survey.Select{
		Message: formatMessage(message),
		Options: choices,
		Default: response,
	}, &response, nil)
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return response, nil
}

// Confirm prompts user for yes or no response.
func (p *Prompt) Confirm(message string) (bool, *failures.Failure) {
	var resp bool
	err := survey.AskOne(&survey.Confirm{
		Message: message,
	}, &resp, nil)
	if err != nil {
		return false, failures.FailUserInput.Wrap(err)
	}
	return resp, nil
}

// Input calls generic prompter Input
func Input(message string, response string) (string, *failures.Failure) {
	resp, fail := prompter.Input(message, response)
	if fail != nil {
		return "", fail
	}
	return resp, nil
}

// Select calls generic prompter Select
func Select(message string, choices []string, response string) (string, *failures.Failure) {
	resp, fail := prompter.Select(message, choices, response)
	if fail != nil {
		return "", fail
	}
	return resp, nil
}

// Confirm calls generic prompter Confirm
func Confirm(message string) (bool, *failures.Failure) {
	resp, fail := prompter.Confirm(message)
	if fail != nil {
		return false, fail
	}
	return resp, nil
}
