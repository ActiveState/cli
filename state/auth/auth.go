package auth

import (
	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/spf13/cobra"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:           "auth",
	Description:    "auth_description",
	Run:            Execute,
	RunWithoutAuth: true,

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_state_auth_token",
			Description: "arg_state_auth_token_description",
			Variable:    &Args.Token,
		},
	},
}

// SignupCommand adds a registration sub-command
var SignupCommand = &commands.Command{
	Name:           "signup",
	Description:    "signup_description",
	Run:            ExecuteSignup,
	RunWithoutAuth: true,
}

// LogoutCommand adds the logout sub-command
var LogoutCommand = &commands.Command{
	Name:           "logout",
	Description:    "logout_description",
	Run:            ExecuteLogout,
	RunWithoutAuth: true,
}

// Args hold the arg values passed through the command line
var Args struct {
	Token string
}

func init() {
	Command.Append(SignupCommand)
	Command.Append(LogoutCommand)
}

// Execute runs our command
func Execute(cmd *cobra.Command, args []string) {
	if api.Auth != nil {
		renewOK, err := api.Client.Authentication.GetRenew(nil, api.Auth)
		if err != nil {
			logging.Warningf("Renewing failed: %s", err)
		} else {
			print.Line(locale.T("logged_in_as", map[string]string{
				"Name": renewOK.Payload.User.Username,
			}))
			return
		}
	}

	if Args.Token == "" {
		plainAuth()
	} else {
		tokenAuth()
	}
}

// ExecuteSignup runs the signup command
func ExecuteSignup(cmd *cobra.Command, args []string) {
	signup()
}

// ExecuteLogout runs the logout command
func ExecuteLogout(cmd *cobra.Command, args []string) {
	api.RemoveAuth()

	print.Line(locale.T("logged_out"))
}
