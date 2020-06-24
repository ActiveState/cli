package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/auth"
)

func newAuthCommand(globals *globalOptions) *captain.Command {
	authRunner := auth.NewAuth()

	params := auth.AuthParams{}

	return captain.NewCommand(
		"auth",
		locale.T("auth_description"),
		[]*captain.Flag{
			{
				Name:        "token",
				Shorthand:   "",
				Description: locale.T("flag_state_auth_token_description"),
				Value:       &params.Token,
			},
			{
				Name:        "username",
				Shorthand:   "",
				Description: locale.T("flag_state_auth_username_description"),
				Value:       &params.Username,
			},
			{
				Name:        "password",
				Shorthand:   "",
				Description: locale.T("flag_state_auth_password_description"),
				Value:       &params.Password,
			},
			{
				Name:        "totp",
				Shorthand:   "",
				Description: locale.T("flag_state_auth_totp_description"),
				Value:       &params.Totp,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			params.Output = globals.Output

			return authRunner.Run(&params)
		},
	)
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
