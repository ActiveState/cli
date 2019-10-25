package auth

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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
		&commands.Flag{
			Name:        "json",
			Description: "flag_json_desc",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.JSON,
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
	JSON     bool
}

func init() {
	Command.Append(SignupCommand)
	Command.Append(LogoutCommand)
}

// Execute runs our command
func Execute(cmd *cobra.Command, args []string) {
	auth := authentication.Get()
	var user []byte
	var fail *failures.Failure
	if auth.Authenticated() {
		if Flags.JSON {
			user, fail = userToJSON(auth.WhoAmI())
			if fail != nil {
				failures.Handle(fail, locale.T("login_err_output"))
				return
			}
			print.Line(string(user))
		} else {
			print.Line(locale.T("logged_in_as", map[string]string{
				"Name": auth.WhoAmI(),
			}))
		}

		return
	}

	if Flags.Token == "" {
		authlet.AuthenticateWithInput(Flags.Username, Flags.Password)
	} else {
		tokenAuth()
	}

	if Flags.JSON {
		user, fail := userToJSON(auth.WhoAmI())
		if fail != nil {
			failures.Handle(fail, locale.T("login_err_output"))
			return
		}
		print.Line(string(user))
	} else {
		print.Line(locale.T("login_success_welcome_back", map[string]string{
			"Name": auth.WhoAmI(),
		}))
	}
}

func userToJSON(username string) ([]byte, *failures.Failure) {
	type userJSON struct {
		Username        string `json:"username,omitempty"`
		Tier            string `json:"tier,omitempty"`
		PrivateProjects bool   `json:"privateProjects"`
	}

	organization, fail := model.FetchOrgByURLName(username)
	if fail != nil {
		return nil, fail
	}

	tiers, fail := model.FetchTiers()
	if fail != nil {
		return nil, fail
	}

	tier := organization.Tier
	privateProjects := false
	for _, t := range tiers {
		privateProjects = (tier == t.Name && t.RequiresPayment)
	}

	userJ := userJSON{username, tier, privateProjects}
	bs, err := json.Marshal(userJ)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return bs, nil
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
