package prompt

import (
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

type EventDispatcher interface {
	EventWithLabel(category, action string, label string, dim ...*dimensions.Values)
}

// Prompter is the interface used to run our prompt from
type Prompter interface {
	Input(title, message string, defaultResponse *string, forcedResponse *string, flags ...ValidatorFlag) (string, error)
	InputAndValidate(title, message string, defaultResponse *string, forcedResponse *string, validator ValidatorFunc, flags ...ValidatorFlag) (string, error)
	Select(title, message string, choices []string, defaultResponse *string, forcedResponse *string) (string, error)
	Confirm(title, message string, defaultChoice *bool, forcedChoice *bool) (bool, error)
	InputSecret(title, message string, flags ...ValidatorFlag) (string, error)
	IsInteractive() bool
	SetInteractive(bool)
	SetForce(bool)
}

// ValidatorFunc is a function pass to the Prompter to perform validation
// on the users input
type ValidatorFunc = survey.Validator

var _ Prompter = &Prompt{}

// Prompt is our main prompting struct
type Prompt struct {
	out           output.Outputer
	analytics     EventDispatcher
	isInteractive bool
	isForced      bool
}

// New creates a new prompter
func New(an EventDispatcher) Prompter {
	return &Prompt{output.Get(), an, true, false}
}

// IsInteractive checks if the prompts can be interactive or should just return default values
func (p *Prompt) IsInteractive() bool {
	return p.isInteractive
}

func (p *Prompt) SetInteractive(interactive bool) {
	p.isInteractive = interactive
}

// SetForce enables prompts to return the force value (which is often different from the
// non-interactive value).
func (p *Prompt) SetForce(force bool) {
	p.isForced = force
}

func (p *Prompt) IsForced() bool {
	return p.isForced
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
func (p *Prompt) Input(title, message string, defaultResponse *string, forcedResponse *string, flags ...ValidatorFlag) (string, error) {
	return p.InputAndValidate(title, message, defaultResponse, forcedResponse, func(val interface{}) error {
		return nil
	}, flags...)
}

// interactiveInputError returns the proper input error for a non-interactive prompt.
// If the terminal cannot show prompts (e.g. Git Bash on Windows), the error mentions this.
// Otherwise, the error simply states the prompt cannot be resolved in non-interactive mode.
// The "message" argument is the prompt's user-facing message.
func interactiveInputError(message string) error {
	if runtime.GOOS == "windows" {
		return locale.NewExternalError("err_non_interactive_mode")
	}
	return locale.NewExternalError("err_non_interactive_prompt", message)
}

// InputAndValidate prompts an input field and allows you to specfiy a custom validation function as well as the built in flags
func (p *Prompt) InputAndValidate(title, message string, defaultResponse *string, forcedResponse *string, validator ValidatorFunc, flags ...ValidatorFlag) (string, error) {
	if p.isForced {
		response := forcedResponse
		if response == nil {
			response = defaultResponse
		}
		if response != nil {
			p.out.Notice(locale.Tr("prompt_using_force", *response))
			return *response, nil
		}
		return "", errs.New("No force option given for forced prompt")
	}

	if !p.isInteractive {
		if defaultResponse != nil {
			p.out.Notice(locale.Tr("prompt_using_non_interactive", *defaultResponse))
			return *defaultResponse, nil
		}
		return "", interactiveInputError(message)
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
		p.out.Notice(output.Emphasize(title))
	}

	// We handle defaults more clearly than the survey package can
	if defaultResponse != nil && *defaultResponse != "" {
		v, err := p.Select("", formatMessage(message, !p.out.Config().Colored), []string{*defaultResponse, locale.Tl("prompt_custom", "Other ..")}, defaultResponse, forcedResponse)
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
func (p *Prompt) Select(title, message string, choices []string, defaultChoice *string, forcedChoice *string) (string, error) {
	if p.isForced {
		choice := forcedChoice
		if choice == nil {
			choice = defaultChoice
		}
		if choice != nil {
			p.out.Notice(locale.Tr("prompt_using_force", *choice))
			return *choice, nil
		}
		return "", errs.New("No force option given for forced prompt")
	}

	if !p.isInteractive {
		if defaultChoice != nil {
			p.out.Notice(locale.Tr("prompt_using_non_interactive", *defaultChoice))
			return *defaultChoice, nil
		}
		return "", interactiveInputError(message)
	}

	if title != "" {
		p.out.Notice(output.Emphasize(title))
	}

	var defChoice string
	if defaultChoice != nil {
		defChoice = *defaultChoice
	}

	var response string
	err := survey.AskOne(&Select{&survey.Select{
		Message:  formatMessage(message, !p.out.Config().Colored),
		Options:  choices,
		Default:  defChoice,
		FilterFn: func(input string, choices []string) []string { return choices }, // no filter
	}}, &response, nil)
	if err != nil {
		return "", locale.NewInputError(err.Error())
	}
	return response, nil
}

// Confirm prompts user for yes or no response.
func (p *Prompt) Confirm(title, message string, defaultChoice *bool, forcedChoice *bool) (bool, error) {
	if p.isForced {
		choice := forcedChoice
		if choice == nil {
			choice = defaultChoice
		}
		if choice != nil {
			p.out.Notice(locale.T("prompt_continue_force"))
			return *choice, nil
		}
		return false, errs.New("No force option given for forced prompt")
	}

	if !p.isInteractive {
		if defaultChoice != nil {
			if *defaultChoice {
				p.out.Notice(locale.T("prompt_continue_non_interactive"))
				return true, nil
			}
			return false, locale.NewInputError("prompt_abort_non_interactive")
		}
		return false, interactiveInputError(message)
	}
	if title != "" {
		p.out.Notice(output.Emphasize(title))
	}

	p.analytics.EventWithLabel(constants.CatPrompt, title, "present")

	var defChoice bool
	if defaultChoice != nil {
		defChoice = *defaultChoice
	}

	var resp bool
	err := survey.AskOne(&Confirm{&survey.Confirm{
		Message: formatMessage(strings.TrimSuffix(message, "\n"), !p.out.Config().Colored),
		Default: defChoice,
	}}, &resp, nil)
	if err != nil {
		if err == terminal.InterruptErr {
			p.analytics.EventWithLabel(constants.CatPrompt, title, "interrupt")
		}
		return false, locale.NewInputError(err.Error())
	}
	p.analytics.EventWithLabel(constants.CatPrompt, title, translateConfirm(resp))

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
		return "", interactiveInputError(message)
	}
	var response string
	validators, err := processValidators(flags)
	if err != nil {
		return "", err
	}

	if title != "" {
		p.out.Notice(output.Emphasize(title))
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
