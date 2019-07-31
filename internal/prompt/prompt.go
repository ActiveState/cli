package prompt

import (
	"fmt"
	"os"

	"github.com/tcnksm/go-input"
	survey "gopkg.in/AlecAivazis/survey.v1"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
)

// Prompter is the interface used to run our prompt from, useful for mocking in tests
type Prompter interface {
	Input(message, defaultResponse string, flags ...ValidatorFlag) (string, *failures.Failure)
	Select(message string, choices []string, defaultResponse string) (string, *failures.Failure)
	Confirm(message string, defaultChoice bool) (bool, *failures.Failure)
	InputSecret(message string, flags ...ValidatorFlag) (string, *failures.Failure)
}

// FailPromptUnknownValidator handles unknown validator erros
var FailPromptUnknownValidator = failures.Type("prompt.unknownvalidator")

// ValidatorFunc is a function pass to the Prompter to perform validation
// on the users input
type ValidatorFunc = func(ans interface{}) error

// Prompt is our main promptig struct
type Prompt struct{}

// New creates a new prompter
func New() Prompter {
	return &Prompt{}
}

// ValidatorFlag represents flags for prompt functions to change their behavior on.
type ValidatorFlag int

const (
	// InputRequired requires that the user provide input
	InputRequired ValidatorFlag = iota
	// IsAlpha
	// IsNumber
	// etc.
)

func newUI() *input.UI {
	return &input.UI{
		Writer: os.Stdout,
		Reader: os.Stdin,
	}
}

// Input prompts the user for input.  The user can specify available validation flags to trigger validation of responses
func (p *Prompt) Input(message, defaultResponse string, flags ...ValidatorFlag) (string, *failures.Failure) {
	var response string
	validators, fail := processValidators(flags)
	if fail != nil {
		return "", fail
	}

	response, err := newUI().Ask(formatMessage(message), &input.Options{
		Default: defaultResponse,
		ValidateFunc: func(s string) error {
			for _, validator := range validators {
				if err := validator(s); err != nil {
					return err
				}
			}
			return nil
		},
		HideOrder: true,
	})

	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}

	return response, nil
}

// Select prompts the user to select one entry from multiple choices
func (p *Prompt) Select(message string, choices []string, defaultChoice string) (string, *failures.Failure) {
	var response string
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
	defaultResponse := "Y"
	if defaultChoice == false {
		defaultResponse = "n"
	}
	response, err := newUI().Ask(formatMessage(message)+" [Y/n]", &input.Options{
		Default:  defaultResponse,
		Required: true,
		ValidateFunc: func(s string) error {
			if s != "Y" && s != "n" {
				return fmt.Errorf(locale.T("err_yes_or_no"))
			}

			return nil
		},
		HideOrder: true,
		Loop:      true,
	})

	if err != nil {
		return false, failures.FailUserInput.Wrap(err)
	}

	return response == "Y", nil
}

// InputSecret prompts the user for input and obfuscates the text in stdout.
// Will fail if empty.
func (p *Prompt) InputSecret(message string, flags ...ValidatorFlag) (string, *failures.Failure) {
	var response string
	validators, fail := processValidators(flags)
	if fail != nil {
		return "", fail
	}

	response, err := newUI().Ask(formatMessage(message), &input.Options{
		Required:    true,
		Mask:        true,
		MaskDefault: true,
		ValidateFunc: func(s string) error {
			for _, validator := range validators {
				if err := validator(s); err != nil {
					return err
				}
			}
			return nil
		},
		HideOrder: true,
	})

	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}

	return response, nil
}

// wrapValidators wraps a list of validators in a wrapper function that can be run by the survey package functions
func wrapValidators(validators []ValidatorFunc) ValidatorFunc {
	validator := func(val interface{}) error {
		for _, v := range validators {
			if error := v(val); error != nil {
				return error
			}
		}
		return nil
	}
	return validator
}

// This function seems like overkill right now but the assumption is we'll have more than one built in validator
func processValidators(flags []ValidatorFlag) ([]ValidatorFunc, *failures.Failure) {
	var validators []ValidatorFunc
	var fail *failures.Failure
	for flag := range flags {
		switch ValidatorFlag(flag) {
		case InputRequired:
			validators = append(validators, inputRequired)
		default:
			fail = FailPromptUnknownValidator.New(locale.T("fail_prompt_bad_flag"))
		}
	}
	return validators, fail
}
