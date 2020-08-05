package ppm

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/skratchdot/open-golang/open"
)

// convertAnswerCreate is the answer that the user can choose if they accept to create a virtual environment.  It is re-used at several places.
var convertAnswerCreate string = locale.Tl("ppm_convert_answer_create", "Create Virtual Runtime Environment")

// ConversionFlowPrimeable defines interface needed to initialize a conversion flow
type ConversionFlowPrimeable interface {
	primer.Prompter
	primer.Outputer
	primer.Projecter
}

// NewConversionFlow creates a new conversion flow structure
func NewConversionFlow(prime ConversionFlowPrimeable) *ConversionFlow {
	return &ConversionFlow{
		prime.Prompt(),
		prime.Output(),
		prime.Project(),
	}
}

// ConversionFlow manages a PPM conversion flow
type ConversionFlow struct {
	prompt  prompt.Prompter
	out     output.Outputer
	project *project.Project
}

// StartIfNecessary checks if the user is in a project directory.
// If not, they are asked to create a project, and (in a wizard-kind-of way) informed about the consequences.
func (cf *ConversionFlow) StartIfNecessary() error {
	// start conversion flow only if we cannot find a project file
	if cf.project != nil {
		return nil
	}

	analytics.Event(analytics.CatPpmConversion, "run")
	r, err := cf.runSurvey()
	if err != nil {
		analytics.EventWithLabel(analytics.CatPpmConversion, "error", errs.Join(err, " :: ").Error())
		return locale.WrapError(err, "ppm_conversion_survey_error", "Conversion flow failed.")
	}

	if r == accepted {
		err = cf.createVirtualEnv()
		if err != nil {
			analytics.EventWithLabel(analytics.CatPpmConversion, "error", errs.Join(err, " :: ").Error())
			return locale.WrapError(err, "ppm_conversion_venv_error", "Failed to create a project.")
		}
	}
	analytics.EventWithLabel(analytics.CatPpmConversion, "completed", r.String())
	return nil
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
		locale.Tl("ppm_convert_answer_why", "Why is this necessary? I Just want to manage packages."),
	}
	choice, fail := cf.prompt.Select(locale.Tl(
		"ppm_convert_create_question", "You need to create a runtime environment to proceed.\n"),
		choices, "")
	if fail != nil {
		return canceled, locale.WrapInputError(fail, "err_ppm_convert_interrupt", "Invalid response received.")
	}

	eventChoices := map[string]string{
		choices[0]: "create-virtual-env-1",
		choices[1]: "asked-why",
	}
	analytics.EventWithLabel(analytics.CatPpmConversion, "selection", eventChoices[choice])

	if choice == choices[0] {
		return accepted, nil
	}

	return cf.explainVirtualEnv(false, false)
}

func (cf *ConversionFlow) createVirtualEnv() error {
	err := runbits.Invoke(cf.out, "tutorial", "new-project")
	if err != nil {
		return locale.WrapError(err, "err_ppm_convert_invoke_tutorial", "Errors occurred while invoking State Tool tutorial command.")
	}

	return nil
}

func (cf *ConversionFlow) explainVirtualEnv(alreadySeenStateToolInfo bool, alreadySeenPlatformInfo bool) (conversionResult, error) {
	stateToolInfo := locale.Tl("ppm_convert_why_state_tool_info", "Find out more about the State Tool")
	platformInfo := locale.Tl("ppm_convert_why_platform_info", "Find out more about the ActiveState Platform")
	no := locale.Tl("ppm_convert_why_no", "But I NEED package management on my global install!")
	var choices []string

	// add choice to open State Tool marketing page (if not looked at before)
	if !alreadySeenStateToolInfo {
		choices = append(choices, stateToolInfo)
	}
	// add choice to open Platform marketing page (if not looked at before)
	if !alreadySeenPlatformInfo {
		choices = append(choices, platformInfo)
	}
	// always add choices to create virtual environment and to say no again
	choices = append(choices, convertAnswerCreate, no)
	explanation := locale.T("ppm_convert_explanation")

	// do not repeat the explanation if the function is called a second time
	if alreadySeenPlatformInfo || alreadySeenStateToolInfo {
		explanation = ""
	}
	choice, fail := cf.prompt.Select(explanation, choices, "")

	if fail != nil {
		return canceled, locale.WrapInputError(fail, "err_ppm_convert_info_interrupt", "Invalid response received.")
	}

	eventChoices := map[string]string{
		stateToolInfo:       "show-state-tool-info",
		platformInfo:        "show-platform-info",
		convertAnswerCreate: "create-virtual-env-2",
		no:                  "still-wants-ppm",
	}
	analytics.EventWithLabel(analytics.CatPpmConversion, "selection", eventChoices[choice])

	switch choice {
	case stateToolInfo:
		cf.openInBrowser(locale.Tl("state_tool_info", "State Tool information"), constants.StateToolMarketingPage)
		// ask again
		return cf.explainVirtualEnv(true, alreadySeenPlatformInfo)
	case platformInfo:
		cf.openInBrowser(locale.Tl("platform_info", "ActiveState Platform information"), constants.PlatformMarketingPage)
		// ask again
		return cf.explainVirtualEnv(alreadySeenStateToolInfo, true)
	case convertAnswerCreate:
		return accepted, nil
	case no:
		return cf.wantGlobalPackageManagement()
	}
	return canceled, nil
}

func (cf *ConversionFlow) openInBrowser(what, url string) {
	cf.out.Print(locale.Tl("ppm_convert_open_browser", "Opening {{.V0}} in your browser", what))
	err := open.Run(url)
	if err != nil {
		logging.Error("Could not open %s in browser: %v", url, err)
		cf.out.Error(locale.Tl(
			"ppm_convert_open_browser_fallback",
			"Could not open {{.V0}} in your browser.\nYou can copy and paste the following URL manually in the address line of your browser:\n{{.V1}}",
			what, url,
		))
	}
}

func (cf *ConversionFlow) wantGlobalPackageManagement() (conversionResult, error) {
	ok := locale.Tl("ppm_convert_create_at_last", "Ok, let's set up a virtual runtime environment")
	perlTooling := locale.Tl("ppm_convert_reject", "I'd rather use conventional Perl tooling.")
	choices := []string{ok, perlTooling}
	choice, fail := cf.prompt.Select(
		locale.Tl("ppm_convert_cpan_info", "You can still use conventional Perl tooling like CPAN, CPANM etc. But you will miss out on the added benefits of the ActiveState Platform.\n"),
		choices, "")

	if fail != nil {
		return canceled, locale.WrapInputError(fail, "err_ppm_convert_final_chance_interrupt", "Invalid response received.")
	}

	eventChoices := map[string]string{
		ok:          "create-virtual-env-3",
		perlTooling: "still-wants-perl-tooling",
	}
	analytics.EventWithLabel(analytics.CatPpmConversion, "selection", eventChoices[choice])

	if choice == choices[0] {
		return accepted, nil
	}
	cf.out.Print(locale.Tl("ppm_convert_reject_sorry", "We're sorry we can't help any further. We'd love to hear more about your use case to see if we can better meet your needs. Please consider posting to our forum at {{.V0}}}.", constants.ForumsURL))

	return rejected, nil
}
