package ppm

import (
	"github.com/ActiveState/cli/internal-as/analytics"
	"github.com/skratchdot/open-golang/open"

	anaConsts "github.com/ActiveState/cli/internal-as/analytics/constants"
	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/multilog"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/primer"
	"github.com/ActiveState/cli/internal-as/prompt"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/project"
)

// convertAnswerCreate is the answer that the user can choose if they accept to create a virtual environment.  It is re-used at several places.
var convertAnswerCreate string = locale.Tl("ppm_convert_answer_create", "Create Virtual Runtime Environment")

// ConversionFlowPrimeable defines interface needed to initialize a conversion flow
type ConversionFlowPrimeable interface {
	primer.Prompter
	primer.Outputer
	primer.Projecter
	primer.Analyticer
}

// NewConversionFlow creates a new conversion flow structure
func NewConversionFlow(prime ConversionFlowPrimeable) *ConversionFlow {
	return &ConversionFlow{
		prime.Prompt(),
		prime.Output(),
		prime.Project(),
		prime.Analytics(),
	}
}

// ConversionFlow manages a PPM conversion flow
type ConversionFlow struct {
	prompt    prompt.Prompter
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
}

// StartIfNecessary checks if the user is in a project directory.
// If not, they are asked to create a project, and (in a wizard-kind-of way) informed about the consequences.
func (cf *ConversionFlow) StartIfNecessary() (bool, error) {
	// start conversion flow only if we cannot find a project file
	if cf.project != nil {
		return false, nil
	}

	cf.analytics.Event(anaConsts.CatPpmConversion, "run")
	r, err := cf.runSurvey()
	if err != nil {
		cf.analytics.EventWithLabel(anaConsts.CatPpmConversion, "error", errs.Join(err, " :: ").Error())
		return true, locale.WrapError(err, "ppm_conversion_survey_error", "Conversion flow failed.")
	}

	if r != accepted {
		cf.analytics.EventWithLabel(anaConsts.CatPpmConversion, "completed", r.String())
		return true, locale.NewInputError("ppm_conversion_rejected", "Virtual environment creation cancelled.")
	}

	err = cf.createVirtualEnv()
	if err != nil {
		cf.analytics.EventWithLabel(anaConsts.CatPpmConversion, "error", errs.Join(err, " :: ").Error())
		return true, locale.WrapError(err, "ppm_conversion_venv_error", "Failed to create a project.")
	}
	cf.analytics.EventWithLabel(anaConsts.CatPpmConversion, "completed", r.String())
	return true, nil
}

type conversionResult int

const (
	accepted conversionResult = iota
	rejected
	canceled
)

func (r conversionResult) String() string {
	return []string{"accepted", "rejected", "canceled"}[r]
}

// runSurvey is the entry point to the conversion survey
func (cf *ConversionFlow) runSurvey() (conversionResult, error) {
	choices := []string{
		convertAnswerCreate,
		locale.Tl("ppm_convert_answer_why", "Why is this necessary? I Just want to manage dependencies"),
	}
	choice, err := cf.prompt.Select("", locale.Tt("ppm_convert_create_question"), choices, new(string))
	if err != nil {
		return canceled, locale.WrapInputError(err, "err_ppm_convert_interrupt", "Invalid response received.")
	}

	cf.out.Print("") // Add some space before next prompt

	eventChoices := map[string]string{
		choices[0]: "create-virtual-env-1",
		choices[1]: "asked-why",
	}
	cf.analytics.EventWithLabel(anaConsts.CatPpmConversion, "selection", eventChoices[choice])

	if choice == choices[0] {
		return accepted, nil
	}

	return cf.explainVirtualEnv()
}

func (cf *ConversionFlow) createVirtualEnv() error {
	err := runbits.InvokeSilent("tutorial", "new-project", "--skip-intro", "--language", "perl")
	if err != nil {
		return locale.WrapError(err, "err_ppm_convert_invoke_tutorial", "Errors occurred while invoking State Tool tutorial command.")
	}

	cf.out.Print("") // Add some space before next prompt

	return nil
}

func (cf *ConversionFlow) explainVirtualEnv() (conversionResult, error) {
	no := locale.Tl("ppm_convert_why_no", "Best practices? No thanks")
	var choices []string

	// always add choices to create virtual environment and to say no again
	choices = append(choices, convertAnswerCreate, no)
	explanation := locale.Tt("ppm_convert_explanation")

	choice, err := cf.prompt.Select("", explanation, choices, new(string))
	if err != nil {
		return canceled, locale.WrapInputError(err, "err_ppm_convert_info_interrupt", "Invalid response received.")
	}
	cf.out.Print("") // Add some space before next prompt

	eventChoices := map[string]string{
		convertAnswerCreate: "create-virtual-env-2",
		no:                  "still-wants-ppm",
	}
	cf.analytics.EventWithLabel(anaConsts.CatPpmConversion, "selection", eventChoices[choice])

	switch choice {
	case convertAnswerCreate:
		return accepted, nil
	case no:
		return cf.explainAskFeedback()
	}
	return canceled, nil
}

func (cf *ConversionFlow) openInBrowser(what, url string) {
	cf.out.Print(locale.Tl("ppm_convert_open_browser", "Opening {{.V0}} in your browser", what))
	err := open.Run(url)
	if err != nil {
		multilog.Error("Could not open %s in browser: %v", url, err)
		cf.out.Error(locale.Tr("browser_fallback", what, url))
	}
}

func (cf *ConversionFlow) explainAskFeedback() (conversionResult, error) {
	ok := locale.Tl("ppm_convert_create_at_last", "Ok, let's set up a virtual runtime environment")
	exit := locale.Tl("ppm_convert_reject", "Exit")
	choices := []string{ok, exit}
	choice, err := cf.prompt.Select("", locale.Tt("ppm_convert_ask_feedback", map[string]interface{}{
		"ForumURL": constants.ForumsURL,
	}), choices, new(string))

	if err != nil {
		return canceled, locale.WrapInputError(err, "err_ppm_convert_final_chance_interrupt", "Invalid response received.")
	}

	cf.out.Print("") // Add some space before next prompt

	eventChoices := map[string]string{
		ok:   "create-virtual-env-3",
		exit: "exit",
	}
	cf.analytics.EventWithLabel(anaConsts.CatPpmConversion, "selection", eventChoices[choice])

	if choice == ok {
		return accepted, nil
	}

	return rejected, nil
}
