package ppm

import (
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/skratchdot/open-golang/open"
)

// convertAnswerCreate is the answer that the user can choose if they accept to create a virtual environment.  It is re-used at several places.
var convertAnswerCreate string = locale.Tl("ppm_convert_answer_create", "Create Virtual Runtime Environment")

// analyticsEventFunc is used to send analytics event
type analyticsEventFunc func(string, string, string)

// surveySelectFunc displays a menu with options that the user can select
type surveySelectFunc func(message string, choices []string, defaultResponse string) (string, *failures.Failure)

type conversionFlow struct {
	survey   surveySelectFunc
	out      output.Outputer
	openURI  func(string) error
	visitedC chan string
	once     sync.Once
}

// StartConversionFlowIfNecessary checks if the user is in a project directory.
// If not, they are asked to create a project, and (in a wizard-kind-of way) informed about the consequences.
func (p *Ppm) StartConversionFlowIfNecessary() error {
	// start conversion flow only if we cannot find a project file
	if p.project == nil {
		c := make(chan os.Signal, 1)
		defer close(c)
		signal.Notify(c, os.Interrupt)
		defer signal.Stop(c)
		cf := newConversionFlow(p.prompt.Select, p.out, open.Run)
		defer cf.Close()

		r, err := cf.run(c, analytics.EventWithLabel, func() { os.Exit(1) })

		if r == accepted {
			cf.createVirtualEnv()
		}
		return err
	}
	return nil
}

func newConversionFlow(survey surveySelectFunc, out output.Outputer, openURI func(string) error) *conversionFlow {
	// Note: the buffer length needs to be big enough to hold the maximum number of visited items
	visitedC := make(chan string, 20)
	cf := &conversionFlow{
		survey:   survey,
		out:      out,
		openURI:  openURI,
		visitedC: visitedC,
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

func (cf *conversionFlow) Close() {
	cf.once.Do(func() {
		close(cf.visitedC)
	})
}

func (cf *conversionFlow) SendAnalyticsEvent(eventFunc analyticsEventFunc, action string) {
	visited := strings.Join(cf.Visited(), ",")
	eventFunc("ppm_conversion", action, visited)
}

func (cf *conversionFlow) registerVisit(what string) {
	cf.visitedC <- what
}

func (cf *conversionFlow) Visited() []string {
	cf.Close()
	var res []string

	for v := range cf.visitedC {
		res = append(res, v)
	}
	return res
}

func (cf *conversionFlow) run(c <-chan os.Signal, eventFunc analyticsEventFunc, exitFunc func()) (conversionResult, error) {
	go func() {
		_, signalReceived := <-c
		if signalReceived {
			cf.SendAnalyticsEvent(eventFunc, canceled.String())
			exitFunc()
		}
	}()

	r, err := cf.runSurvey()
	cf.SendAnalyticsEvent(eventFunc, r.String())
	if err != nil {
		return r, err
	}

	return r, nil
}

func (cf *conversionFlow) runSurvey() (conversionResult, error) {
	choices := []string{
		convertAnswerCreate,
		locale.Tl("ppm_convert_answer_why", "Why is this necessary? I Just want to manage packages."),
	}
	choice, fail := cf.survey(locale.Tl(
		"ppm_convert_create_question", "You need to create a runtime environment to proceed.\n"),
		choices, "")
	if fail != nil {
		return canceled, fail.ToError()
	}
	if choice == choices[0] {
		return accepted, nil
	}
	cf.registerVisit("asked why")
	return cf.explainVirtualEnv(0)
}

func (cf *conversionFlow) createVirtualEnv() error {
	// TODO: start wizard

	cf.out.Print(locale.Tl(
		"ppm_convert_after_tutorial",
		"For your convenience you can continue to use ppm commands once you’ve activated your virtual runtime environment. We’ll give you handy tips on how your commands map to State Tool so you can learn as you go.",
	))
	return nil
}

type docSelection int

const (
	stateToolInfoShown docSelection = 1
	platformInfoShown               = 2
)

func (cf *conversionFlow) explainVirtualEnv(prevSelection docSelection) (conversionResult, error) {

	stateToolInfo := locale.Tl("ppm_convert_why_state_tool_info", "Find out more about the State Tool")
	platformInfo := locale.Tl("ppm_convert_why_platform_info", "Find out more about the ActiveState Platform")
	no := locale.Tl("ppm_convert_why_no", "But I NEED package management on my global install!")
	var choices []string
	if prevSelection&stateToolInfoShown == 0 {
		choices = append(choices, stateToolInfo)
	}
	if prevSelection&platformInfoShown == 0 {
		choices = append(choices, platformInfo)
	}
	choices = append(choices, convertAnswerCreate, no)
	explanation := locale.Tl("ppm_convert_explanation", "State Tool was developed from the ground up with modern software development practices in mind. Development environments with globally installed language runtime environments are increasingly shunned by modern development practices, and as a result the State Tool and the ActiveState Platform tries to do away with them entirely.\n")
	if prevSelection != 0 {
		explanation = ""
	}
	choice, fail := cf.survey(explanation, choices, "")
	if fail != nil {
		return canceled, fail.ToError()
	}

	switch choice {
	case stateToolInfo:
		cf.openInBrowser(locale.Tl("state_tool_info", "State Tool information"), constants.StateToolMarketingPage)
		cf.registerVisit("visited state tool info")
		return cf.explainVirtualEnv(prevSelection | stateToolInfoShown)
	case platformInfo:
		cf.openInBrowser(locale.Tl("platform_info", "ActiveState Platform information"), constants.PlatformMarketingPage)
		cf.registerVisit("visited platform info")
		return cf.explainVirtualEnv(prevSelection | platformInfoShown)
	case convertAnswerCreate:
		return accepted, nil
	case no:
		cf.registerVisit("still wanted ppm")
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
		locale.Tl("ppm_convert_cpan_info", "You can still use conventional Perl tooling like CPAN, CPANM etc. But you will miss out on the added benefits of the ActiveState Platform.\n"),
		choices, "")
	if fail != nil {
		return canceled, fail.ToError()
	}
	if choice == choices[0] {
		return accepted, nil
	}
	cf.out.Print(locale.Tl("ppm_convert_reject_sorry", "We're sorry we can't help any further. We'd love to hear more about your use case to see if we can better meet your needs. Please consider posting at {{.V0}} or contacting {{.V1}}", constants.ForumsURL, constants.SupportEmail))

	return rejected, nil
}
