package ppm

import (
	"os"
	"os/exec"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/textutils"
	"github.com/skratchdot/open-golang/open"
)

// convertAnswerCreate is the answer that the user can choose if they accept to create a virtual environment.  It is re-used at several places.
var convertAnswerCreate string = locale.Tl("ppm_convert_answer_create", "Create Virtual Runtime Environment")

// analyticsEventFunc is used to send analytics event
type analyticsEventFunc func(string, string, string)

// surveySelectFunc displays a menu with options that the user can select
type surveySelectFunc func(message string, choices []string, defaultResponse string) (string, *failures.Failure)

const (
	askedWhy          string = "asked-why"
	seenStateToolInfo        = "state-tool-info"
	seenPlatformInfo         = "platform-info"
	notConvinced             = "still-wants-ppm"
)

type conversionFlow struct {
	survey    surveySelectFunc
	out       output.Outputer
	openURI   func(string) error
	eventFunc analyticsEventFunc
}

// StartConversionFlowIfNecessary checks if the user is in a project directory.
// If not, they are asked to create a project, and (in a wizard-kind-of way) informed about the consequences.
func (p *Ppm) StartConversionFlowIfNecessary() error {
	// start conversion flow only if we cannot find a project file
	if p.project == nil {
		cf := newConversionFlow(p.prompt.Select, p.out, open.Run, analytics.EventWithLabel)

		analytics.Event(analytics.CatPpmConversion, "run")
		r, err := cf.runSurvey()
		analytics.EventWithLabel(analytics.CatPpmConversion, "completed", r.String())

		if r == accepted {
			cf.createVirtualEnv()
		}
		return err
	}
	return nil
}

func newConversionFlow(survey surveySelectFunc, out output.Outputer, openURI func(string) error, eventFunc analyticsEventFunc) *conversionFlow {
	cf := &conversionFlow{
		survey:    survey,
		out:       out,
		openURI:   openURI,
		eventFunc: eventFunc,
	}

	return cf
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

// sendClickEvent sends an analytics event if the user selects a menu-item that does not lead to a conclusive result
func (cf *conversionFlow) sendClickEvent(what string) {
	cf.eventFunc(analytics.CatPpmConversion, "click", what)
}

// runSurvey is the entry point to the conversion survey
func (cf *conversionFlow) runSurvey() (conversionResult, error) {
	choices := []string{
		convertAnswerCreate,
		locale.Tl("ppm_convert_answer_why", "Why is this necessary? I Just want to manage packages."),
	}
	choice, fail := cf.survey(textutils.WordWrap(locale.Tl(
		"ppm_convert_create_question", "You need to create a runtime environment to proceed.\n")),
		choices, "")
	if fail != nil {
		return canceled, locale.WrapInputError(fail, "err_ppm_convert_interrupt", "Invalid response received.")
	}
	if choice == choices[0] {
		return accepted, nil
	}

	cf.sendClickEvent(askedWhy)
	return cf.explainVirtualEnv(false, false)
}

func (cf *conversionFlow) createVirtualEnv() error {
	exe, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_ppm_convert_invoke_exe", "Could not detect executable path of State Tool.")
	}

	cmd := exec.Command(exe, "tutorial", "new-project")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = cmd.Run()
	if err != nil {
		return locale.WrapError(err, "err_ppm_convert_invoke_tutorial", "Errors occurred while invoking State Tool tutorial command.")
	}

	// print a new line to separate from the last tutorial message
	cf.out.Print("\n")
	// sleep for a second to give a visual feedback that we have returned to the conversion flow
	time.Sleep(1 * time.Second)

	cf.out.Print(textutils.WordWrap(locale.Tl(
		"ppm_convert_after_tutorial",
		"For your convenience you can continue to use ppm commands once you’ve activated your virtual runtime environment. We’ll give you handy tips on how your commands map to State Tool so you can learn as you go.",
	)))
	return nil
}

func (cf *conversionFlow) explainVirtualEnv(alreadySeenStateToolInfo bool, alreadySeenPlatformInfo bool) (conversionResult, error) {
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
	explanation := textutils.WordWrap(locale.Tl("ppm_convert_explanation", "State Tool was developed from the ground up with modern software development practices in mind. Development environments with globally installed language runtime environments are increasingly shunned by modern development practices, and as a result the State Tool and the ActiveState Platform tries to do away with them entirely.\n"))

	// do not repeat the explanation if the function is called a second time
	if alreadySeenPlatformInfo || alreadySeenStateToolInfo {
		explanation = ""
	}
	choice, fail := cf.survey(explanation, choices, "")

	if fail != nil {
		return canceled, locale.WrapInputError(fail, "err_ppm_convert_info_interrupt", "Invalid response received.")
	}

	switch choice {
	case stateToolInfo:
		cf.sendClickEvent(stateToolInfo)
		cf.openInBrowser(locale.Tl("state_tool_info", "State Tool information"), constants.StateToolMarketingPage)
		// ask again
		return cf.explainVirtualEnv(true, alreadySeenPlatformInfo)
	case platformInfo:
		cf.sendClickEvent(platformInfo)
		cf.openInBrowser(locale.Tl("platform_info", "ActiveState Platform information"), constants.PlatformMarketingPage)
		// ask again
		return cf.explainVirtualEnv(alreadySeenStateToolInfo, true)
	case convertAnswerCreate:
		return accepted, nil
	case no:
		cf.sendClickEvent(notConvinced)
		return cf.wantGlobalPackageManagement()
	}
	return canceled, nil
}

func (cf *conversionFlow) openInBrowser(what, url string) {
	cf.out.Print(locale.Tl("ppm_convert_open_browser", "Opening {{.V0}} in your browser", what))
	err := cf.openURI(url)
	if err != nil {
		logging.Error("Could not open %s in browser: %v", url, err)
		cf.out.Error(locale.Tl(
			"ppm_convert_open_browser_fallback",
			"Could not open {{.V0}} in your browser.\nYou can copy and paste the following URL manually in the address line of your browser:\n{{.V1}}",
			what, url,
		))
	}
}

func (cf *conversionFlow) wantGlobalPackageManagement() (conversionResult, error) {
	choices := []string{
		locale.Tl("ppm_convert_create_at_last", "Ok, let's set up a virtual runtime environment"),
		locale.Tl("ppm_convert_reject", "I'd rather use conventional Perl tooling."),
	}
	choice, fail := cf.survey(
		textutils.WordWrap(locale.Tl("ppm_convert_cpan_info", "You can still use conventional Perl tooling like CPAN, CPANM etc. But you will miss out on the added benefits of the ActiveState Platform.\n")),
		choices, "")

	if fail != nil {
		return canceled, locale.WrapInputError(fail, "err_ppm_convert_final_chance_interrupt", "Invalid response received.")
	}
	if choice == choices[0] {
		return accepted, nil
	}
	cf.out.Print(textutils.WordWrap(locale.Tl("ppm_convert_reject_sorry", "We're sorry we can't help any further. We'd love to hear more about your use case to see if we can better meet your needs. Please consider posting to our forum at {{.V0}}}.", constants.ForumsURL)))

	return rejected, nil
}
