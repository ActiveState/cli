package prompt

import (
	"github.com/ActiveState/cli/internal/failures"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Prompter is the interface used to run our prompt from, useful for mocking in tests
type Prompter interface {
	Input(message string, response string) (string, *failures.Failure)
	Select(message string, choices []string, response string) (string, *failures.Failure)
}

// Prompt is our main promptig struct
type Prompt struct{}

// New creates a new prompter
func New() Prompter {
	return &Prompt{}
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
