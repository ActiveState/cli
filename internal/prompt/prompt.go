package prompt

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

// Prompter is the interface used to run our prompt from, useful for mocking in tests
type Prompter interface {
	Input(title, message string, defaultResponse *string, flags ...ValidatorFlag) (string, error)
	InputAndValidate(title, message string, defaultResponse *string, validator ValidatorFunc, flags ...ValidatorFlag) (string, error)
	Select(title, message string, choices []string, defaultResponse *string) (string, error)
	Confirm(title, message string, defaultChoice *bool) (bool, error)
	InputSecret(title, message string, flags ...ValidatorFlag) (string, error)
	IsInteractive() bool
}

// ValidatorFunc is a function pass to the Prompter to perform validation
// on the users input
type ValidatorFunc = survey.Validator

var _ Prompter = &Prompt{}

// Prompt is our main prompting struct
type Prompt struct {
	out           output.Outputer
	analytics     analytics.AnalyticsDispatcher
	isInteractive bool
}

// New creates a new prompter
func New(isInteractive bool, an analytics.AnalyticsDispatcher) Prompter {
	return &Prompt{output.Get(), an, isInteractive}
}

// IsInteractive checks if the prompts can be interactive or should just return default values
func (p *Prompt) IsInteractive() bool {
	return p.isInteractive
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
func (p *Prompt) Input(title, message string, defaultResponse *string, flags ...ValidatorFlag) (string, error) {
	return p.InputAndValidate(title, message, defaultResponse, func(val interface{}) error {
		return nil
	}, flags...)
}

// InputAndValidate prompts an input field and allows you to specfiy a custom validation function as well as the built in flags
func (p *Prompt) InputAndValidate(title, message string, defaultResponse *string, validator ValidatorFunc, flags ...ValidatorFlag) (string, error) {
	if !p.isInteractive {
		if defaultResponse != nil {
			logging.Debug("Selecting default choice %s for Input prompt %s in non-interactive mode", *defaultResponse, title)
			return *defaultResponse, nil
		}
		return "", locale.NewInputError("err_non_interactive_prompt", message)
	}

	var response string
	flagValidators, err := processValidators(flags)
	if err != nil {
		return "", err
	}
	if len(flagValidators) != 0 {
		validator = wrapValidators(append(flagValidators, validator))
	}

	if title != "" {
		p.out.Notice(output.SubHeading(title))
	}

	// We handle defaults more clearly than the survey package can
	if defaultResponse != nil && *defaultResponse != "" {
		v, err := p.Select("", formatMessage(message, !p.out.Config().Colored), []string{*defaultResponse, locale.Tl("prompt_custom", "Other ..")}, defaultResponse)
		if err != nil {
			return "", err
		}
		if v == *defaultResponse {
			return v, nil
		}
		message = ""
	}

	err = survey.AskOne(&Input{&survey.Input{
		Message: formatMessage(message, !p.out.Config().Colored),
	}}, &response, validator)
	if err != nil {
		return "", locale.NewInputError(err.Error())
	}

	return response, nil
}

// Select prompts the user to select one entry from multiple choices
func (p *Prompt) Select(title, message string, choices []string, defaultChoice *string) (string, error) {
	if !p.isInteractive {
		if defaultChoice != nil {
			logging.Debug("Selecting default choice %s for Select prompt %s in non-interactive mode", *defaultChoice, title)
			return *defaultChoice, nil
		}
		return "", locale.NewInputError("err_non_interactive_prompt", message)
	}

	if title != "" {
		p.out.Notice(output.SubHeading(title))
	}

	var defChoice string
	if defaultChoice != nil {
		defChoice = *defaultChoice
	}

	var response string
	err := survey.AskOne(&Select{&survey.Select{
		Message: formatMessage(message, !p.out.Config().Colored),
		Options: choices,
		Default: defChoice,
	}}, &response, nil)
	if err != nil {
		return "", locale.NewInputError(err.Error())
	}
	return response, nil
}

// Confirm prompts user for yes or no response.
func (p *Prompt) Confirm(title, message string, defaultChoice *bool) (bool, error) {
	if !p.isInteractive {
		if defaultChoice != nil {
			logging.Debug("Prompt %s confirmed with default choice %v in non-interactive mode", title, defaultChoice)
			return *defaultChoice, nil
		}
		return false, locale.NewInputError("err_non_interactive_prompt", message)
	}
	if title != "" {
		p.out.Notice(output.SubHeading(title))
	}

	p.analytics.EventWithLabel(analytics.CatPrompt, title, "present")

	var defChoice bool
	if defaultChoice != nil {
		defChoice = *defaultChoice
	}

	var resp bool
	err := survey.AskOne(&Confirm{&survey.Confirm{
		Message: formatMessage(message, !p.out.Config().Colored),
		Default: defChoice,
	}}, &resp, nil)
	if err != nil {
		if err == terminal.InterruptErr {
			p.analytics.EventWithLabel(analytics.CatPrompt, title, "interrupt")
		}
		return false, locale.NewInputError(err.Error())
	}
	p.analytics.EventWithLabel(analytics.CatPrompt, title, translateConfirm(resp))

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
func (p *Prompt) InputSecret(title, message string, flags ...ValidatorFlag) (string, error) {
	if !p.isInteractive {
		return "", locale.NewInputError("err_non_interactive_prompt", message)
	}
	var response string
	validators, err := processValidators(flags)
	if err != nil {
		return "", err
	}

	if title != "" {
		p.out.Notice(output.SubHeading(title))
	}

	err = survey.AskOne(&Password{&survey.Password{
		Message: formatMessage(message, !p.out.Config().Colored),
	}}, &response, wrapValidators(validators))
	if err != nil {
		return "", locale.NewInputError(err.Error())
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
func processValidators(flags []ValidatorFlag) ([]ValidatorFunc, error) {
	var validators []ValidatorFunc
	var err error
	for flag := range flags {
		switch ValidatorFlag(flag) {
		case InputRequired:
			validators = append(validators, inputRequired)
		default:
			err = locale.NewError("err_prompt_bad_flag")
		}
	}
	return validators, err
}
