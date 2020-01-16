package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/auth"
)

type AuthOpts struct {
	Token    string
	Username string
	Password string
}

func newAuthCommand(globals *globalOptions) *captain.Command {
	authRunner := auth.NewAuth()

	opts := AuthOpts{}
	return captain.NewCommand(
		"auth",
		locale.T("auth_description"),
		[]*captain.Flag{
			{
				Name:        "token",
				Shorthand:   "",
				Description: locale.T("arg_state_auth_token_description"),
				Value:       &opts.Token,
			},
			{
				Name:        "username",
				Shorthand:   "",
				Description: locale.T("arg_state_auth_username_description"),
				Value:       &opts.Username,
			},
			{
				Name:        "password",
				Shorthand:   "",
				Description: locale.T("arg_state_auth_password_description"),
				Value:       &opts.Password,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return authRunner.Run(newAuthRunParams(opts, globals))
		},
	)
}

func newAuthRunParams(opts AuthOpts, globals *globalOptions) *auth.AuthParams {
	return &auth.AuthParams{
		Output:   globals.Output,
		Token:    opts.Token,
		Username: opts.Username,
		Password: opts.Password,
	}
}

func newSignupCommand() *captain.Command {
	signupRunner := auth.NewSignup()
	return captain.NewCommand(
		"signup",
		locale.T("signup_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return signupRunner.Run()
		},
	)
}

func newLogoutCommand() *captain.Command {
	logoutRunner := auth.NewLogout()
	return captain.NewCommand(
		"logout",
		locale.T("logout_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return logoutRunner.Run()
		},
	)
}
