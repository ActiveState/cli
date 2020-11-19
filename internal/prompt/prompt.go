package prompt

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

// Prompter is the interface used to run our prompt from, useful for mocking in tests
type Prompter interface {
	Input(title, message, defaultResponse string, flags ...ValidatorFlag) (string, *failures.Failure)
	InputAndValidate(title, message, defaultResponse string, validator ValidatorFunc, flags ...ValidatorFlag) (string, *failures.Failure)
	Select(title, message string, choices []string, defaultResponse string) (string, *failures.Failure)
	Confirm(title, message string, defaultChoice bool) (bool, *failures.Failure)
	InputSecret(title, message string, flags ...ValidatorFlag) (string, *failures.Failure)
}

// FailPromptUnknownValidator handles unknown validator erros
var FailPromptUnknownValidator = failures.Type("prompt.unknownvalidator")

// ValidatorFunc is a function pass to the Prompter to perform validation
// on the users input
type ValidatorFunc = survey.Validator

// Prompt is our main promptig struct
type Prompt struct {
	out output.Outputer
}

// New creates a new prompter
func New() Prompter {
	return &Prompt{output.Get()}
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

// Input prompts the user for input.  The user can specify available validation flags to trigger validation of responses
func (p *Prompt) Input(title, message, defaultResponse string, flags ...ValidatorFlag) (string, *failures.Failure) {
	return p.InputAndValidate(title, message, defaultResponse, func(val interface{}) error {
		return nil
	}, flags...)
}

// InputAndValidate prompts an input field and allows you to specfiy a custom validation function as well as the built in flags
func (p *Prompt) InputAndValidate(title, message, defaultResponse string, validator ValidatorFunc, flags ...ValidatorFlag) (string, *failures.Failure) {
	var response string
	flagValidators, fail := processValidators(flags)
	if fail != nil {
		return "", fail
	}
	if len(flagValidators) != 0 {
		validator = wrapValidators(append(flagValidators, validator))
	}

	if title != "" {
		p.out.Notice(output.SubHeading(title))
	}

	// We handle defaults more clearly than the survey package can
	if defaultResponse != "" {
		v, fail := p.Select("", formatMessage(message, !p.out.Config().Colored), []string{defaultResponse, locale.Tl("prompt_custom", "Other ..")}, defaultResponse)
		if fail != nil {
			return "", fail
		}
		if v == defaultResponse {
			return v, nil
		}
		message = ""
	}

	err := survey.AskOne(&Input{&survey.Input{
		Message: formatMessage(message, !p.out.Config().Colored),
	}}, &response, validator)
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}

	return response, nil
}

// Select prompts the user to select one entry from multiple choices
func (p *Prompt) Select(title, message string, choices []string, defaultChoice string) (string, *failures.Failure) {
	if title != "" {
		p.out.Notice(output.SubHeading(title))
	}

	var response string
	err := survey.AskOne(&Select{&survey.Select{
		Message: formatMessage(message, !p.out.Config().Colored),
		Options: choices,
		Default: defaultChoice,
	}}, &response, nil)
	if err != nil {
		return "", failures.FailUserInput.Wrap(err)
	}
	return response, nil
}

// Confirm prompts user for yes or no response.
func (p *Prompt) Confirm(title, message string, defaultChoice bool) (bool, *failures.Failure) {
	if title != "" {
		p.out.Notice(output.SubHeading(title))
	}

	analytics.EventWithLabel(analytics.CatPrompt, title, "present")

	var resp bool
	err := survey.AskOne(&Confirm{&survey.Confirm{
		Message: formatMessage(message, !p.out.Config().Colored),
		Default: defaultChoice,
	}}, &resp, nil)
	if err != nil {
		if err == terminal.InterruptErr {
			analytics.EventWithLabel(analytics.CatPrompt, title, "interrupt")
		}
		return false, failures.FailUserInput.Wrap(err)
	}
	analytics.EventWithLabel(analytics.CatPrompt, title, translateConfirm(resp))

	return resp, nil
}

func translateConfirm(confirm bool) string {
	if confirm {
		return "positive"
	}
	return "negative"
}

// InputSecret prompts the user for input and obfuscates the text in stdout.
// Will fail if empty.
func (p *Prompt) InputSecret(title, message string, flags ...ValidatorFlag) (string, *failures.Failure) {
	var response string
	validators, fail := processValidators(flags)
	if fail != nil {
		return "", fail
	}

	if title != "" {
		p.out.Notice(output.SubHeading(title))
	}

	err := survey.AskOne(&Password{&survey.Password{
		Message: formatMessage(message, !p.out.Config().Colored),
	}}, &response, wrapValidators(validators))
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
