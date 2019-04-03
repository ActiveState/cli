package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/spf13/cobra"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "auth",
	Description: "auth_description",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "token",
			Description: "arg_state_auth_token_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Token,
		},
		&commands.Flag{
			Name:        "username",
			Description: "arg_state_auth_username_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Username,
		},
		&commands.Flag{
			Name:        "password",
			Description: "arg_state_auth_password_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Password,
		},
	},
}

// SignupCommand adds a registration sub-command
var SignupCommand = &commands.Command{
	Name:        "signup",
	Description: "signup_description",
	Run:         ExecuteSignup,
}

// LogoutCommand adds the logout sub-command
var LogoutCommand = &commands.Command{
	Name:        "logout",
	Description: "logout_description",
	Run:         ExecuteLogout,
}

// Flags hold the arg values passed through the command line
var Flags struct {
	Token    string
	Username string
	Password string
}

func init() {
	Command.Append(SignupCommand)
	Command.Append(LogoutCommand)
}

// Execute runs our command
func Execute(cmd *cobra.Command, args []string) {
	if authentication.Get().Authenticated() {
		renewOK, err := authentication.Client().Authentication.GetRenew(nil, authentication.ClientAuth())
		if err != nil {
			logging.Warningf("Renewing failed: %s", err)
		} else {
			print.Line(locale.T("logged_in_as", map[string]string{
				"Name": renewOK.Payload.User.Username,
			}))
			return
		}
	}

	if Flags.Token == "" {
		authlet.AuthenticateWithInput(Flags.Username, Flags.Password)
	} else {
		tokenAuth()
	}
}

// ExecuteSignup runs the signup command
func ExecuteSignup(cmd *cobra.Command, args []string) {
	authlet.Signup()
}

// ExecuteLogout runs the logout command
func ExecuteLogout(cmd *cobra.Command, args []string) {
	doLogout()
	print.Line(locale.T("logged_out"))
}

func doLogout() {
	authentication.Logout()
	keypairs.DeleteWithDefaults()
}
