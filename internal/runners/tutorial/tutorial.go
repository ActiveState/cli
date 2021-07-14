package tutorial

import (
	"fmt"
	"os"

	"github.com/skratchdot/open-golang/open"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Tutorial struct {
	outputer output.Outputer
	auth     *authentication.Auth
	prompt   prompt.Prompter
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Auther
	primer.Configurer
}

func New(primer primeable) *Tutorial {
	return &Tutorial{primer.Output(), primer.Auth(), primer.Prompt()}
}

type NewProjectParams struct {
	SkipIntro bool
	Language  language.Language
}

func (t *Tutorial) RunNewProject(params NewProjectParams) error {
	analytics.EventWithLabel(analytics.CatTutorial, "run", fmt.Sprintf("skipIntro=%v,language=%v", params.SkipIntro, params.Language.String()))

	// Print intro
	if !params.SkipIntro {
		t.outputer.Print(locale.Tt("tutorial_newproject_intro"))
	}

	// Prompt for authentication
	if !t.auth.Authenticated() {
		if err := t.authFlow(); err != nil {
			return err
		}
	}

	// Prompt for language
	lang := params.Language
	if lang == language.Unset {
		choice, err := t.prompt.Select(
			"",
			locale.Tl("tutorial_language", "What language would you like to use for your new virtual environment?"),
			[]string{language.Perl.Text(), language.Python3.Text(), language.Python2.Text()},
			new(string),
		)
		if err != nil {
			return locale.WrapInputError(err, "err_tutorial_prompt_language", "Invalid response received.")
		}
		lang = language.MakeByText(choice)
		if lang == language.Unknown || lang == language.Unset {
			return locale.NewError("err_tutorial_language_unknown", "Invalid language selected: {{.V0}}.", choice)
		}
		analytics.EventWithLabel(analytics.CatTutorial, "choose-language", lang.String())
	}

	// Prompt for project name
	defProjectInput := lang.Text()
	name, err := t.prompt.Input("", locale.Tl("tutorial_prompt_projectname", "What do you want to name your project?"), &defProjectInput)
	if err != nil {
		return locale.WrapInputError(err, "err_tutorial_prompt_projectname", "Invalid response received.")
	}

	// Prompt for project dir
	homeDir, _ := fileutils.HomeDir()
	dir, err := t.prompt.Input("", locale.Tl(
		"tutorial_prompt_projectdir",
		"Where would you like your project directory to be mapped? This is usually the root of your repository, or the place where you have your project dotfiles."), &homeDir)
	if err != nil {
		return locale.WrapInputError(err, "err_tutorial_prompt_projectdir", "Invalid response received.")
	}

	// Create dir and switch to it
	if err := fileutils.MkdirUnlessExists(dir); err != nil {
		return locale.WrapInputError(err, "err_tutorial_mkdir", "Could not create directory: {{.V0}}.", dir)
	}
	if err := os.Chdir(dir); err != nil {
		return locale.WrapInputError(err, "err_tutorial_chdir", "Could not change directory to: {{.V0}}", dir)
	}

	// Run state init
	if err := runbits.Invoke(t.outputer, "init", t.auth.WhoAmI()+"/"+name, lang.String(), "--path", dir); err != nil {
		return locale.WrapInputError(err, "err_tutorial_state_init", "Could not initialize project.")
	}

	// Run state push
	if err := runbits.Invoke(t.outputer, "push"); err != nil {
		return locale.WrapInputError(err, "err_tutorial_state_push", "Could not push project to ActiveState Platform, try manually running `state push` from your project directory at {{.V0}}.", dir)
	}

	// Print outro
	t.outputer.Print(locale.Tt(
		"tutorial_newproject_outro", map[string]interface{}{
			"Dir":  dir,
			"URL":  fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, t.auth.WhoAmI(), name),
			"Docs": constants.DocumentationURL,
		}))

	return nil
}

// authFlow is invoked when the user is not authenticated, it will prompt for sign in or sign up
func (t *Tutorial) authFlow() error {
	analytics.Event(analytics.CatTutorial, "authentication-flow")

	// Sign in / Sign up choices
	signIn := locale.Tl("tutorial_signin", "Sign In")
	signUpCLI := locale.Tl("tutorial_createcli", "Create Account via Command Line")
	signUpBrowser := locale.Tl("tutorial_createbrowser", "Create Account via Browser")
	choices := []string{signIn, signUpCLI, signUpBrowser}

	// Prompt for auth
	choice, err := t.prompt.Select(
		"",
		locale.Tl("tutorial_need_account", "In order to create a virtual environment you must have an ActiveState Platform account"),
		choices,
		&signIn,
	)
	if err != nil {
		return locale.WrapInputError(err, "err_tutorial_prompt_account", "Invalid response received.")
	}

	// Evaluate user selection
	switch choice {
	case signIn:
		analytics.EventWithLabel(analytics.CatTutorial, "authentication-action", "sign-in")
		if err := runbits.Invoke(t.outputer, "auth"); err != nil {
			return locale.WrapInputError(err, "err_tutorial_signin", "Sign in failed. You could try manually signing in by running `state auth`.")
		}
	case signUpCLI:
		analytics.EventWithLabel(analytics.CatTutorial, "authentication-action", "sign-up")
		if err := runbits.Invoke(t.outputer, "auth", "signup"); err != nil {
			return locale.WrapInputError(err, "err_tutorial_signup", "Sign up failed. You could try manually signing up by running `state auth signup`.")
		}
	case signUpBrowser:
		analytics.EventWithLabel(analytics.CatTutorial, "authentication-action", "sign-up-browser")
		err := open.Run(constants.PlatformSignupURL)
		if err != nil {
			return locale.WrapInputError(err, "err_tutorial_browser", "Could not open browser, please manually navigate to {{.V0}}.", constants.PlatformSignupURL)
		}
		t.outputer.Notice(locale.Tl("tutorial_signing_ready", "[NOTICE]Please sign in once you have finished signing up via your browser.[/RESET]"))
		if err := runbits.Invoke(t.outputer, "auth"); err != nil {
			return locale.WrapInputError(err, "err_tutorial_signin", "Sign in failed. You could try manually signing in by running `state auth`.")
		}
	}

	if err := t.auth.Authenticate(); err != nil {
		return locale.WrapError(err, "err_tutorial_auth", "Could not authenticate after invoking `state auth ..`.")
	}

	analytics.Event(analytics.CatTutorial, "authentication-flow-complete")

	return nil
}
