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

func newAuthCommand() *captain.Command {
	authRunner := auth.NewAuth()

	opts := AuthOpts{}
	authCmd := captain.NewCommand(
		"auth",
		locale.T("auth_description"),
		[]*captain.Flag{
			{
				Name:        "token",
				Shorthand:   "",
				Description: locale.T("arg_state_auth_token_description"),
				Type:        captain.TypeString,
				StringVar:   &opts.Token,
			},
			{
				Name:        "username",
				Shorthand:   "",
				Description: locale.T("arg_state_auth_username_description"),
				Type:        captain.TypeString,
				StringVar:   &opts.Username,
			},
			{
				Name:        "password",
				Shorthand:   "",
				Description: locale.T("arg_state_auth_password_description"),
				Type:        captain.TypeString,
				StringVar:   &opts.Password,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return authRunner.Run(&auth.AuthParams{
				// Output:   globals.Output,
				Token:    opts.Token,
				Username: opts.Username,
				Password: opts.Password,
			})
		},
	)

	authCmd.AddChildren(
		newSignupCommand(),
		newLogoutCommand(),
	)

	return authCmd
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
