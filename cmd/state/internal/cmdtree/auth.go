package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/auth"
)

func newAuthCommand(prime *primer.Values) *captain.Command {
	authRunner := auth.NewAuth(prime)

	params := auth.AuthParams{}

	return captain.NewCommand(
		"auth",
		locale.Tl("auth_title", "Signing In To The ActiveState Platform"),
		locale.T("auth_description"),
		prime.Output(),
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
			return authRunner.Run(&params)
		},
	).SetGroup(PlatformGroup)
}

func newSignupCommand(prime *primer.Values) *captain.Command {
	signupRunner := auth.NewSignup(prime)
	return captain.NewCommand(
		"signup",
		locale.Tl("signup_title", "Signing Up With The ActiveState Platform"),
		locale.T("signup_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return signupRunner.Run()
		},
	)
}

func newLogoutCommand(prime *primer.Values) *captain.Command {
	logoutRunner := auth.NewLogout(prime)
	return captain.NewCommand(
		"logout",
		locale.Tl("logout_title", "Logging Out Of The ActiveState Platform"),
		locale.T("logout_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return logoutRunner.Run()
		},
	)
}
