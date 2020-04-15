package invite

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// MaxParallelRequests is the maximum number of invite requests that we want to send in parallel
const MaxParallelRequests = 10

// Command is the organization command's definition.
var Command = &commands.Command{
	Name:        "invite",
	Aliases:     []string{},
	Description: "invite_description",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "organization",
			Description: "invite_flag_organization_description",
			Type:        commands.TypeString,
			StringVar:   &Args.Organization,
		},
		&commands.Flag{
			Name:        "role",
			Description: "invite_flag_role_description",
			Type:        commands.TypeString,
			StringVar:   &Args.RoleString,
		},
	},

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "<email1>,[<email2>,..]",
			Description: "invite_arg_emails",
			Required:    true,
			Variable:    &Args.EmailList,
		},
	},
}

var prompter prompt.Prompter

func init() {
	prompter = prompt.New()
}

// Arguments is a structure for command line parameters and flags
type Arguments struct {
	EmailList    string
	Organization string
	RoleString   string
}

// Args stores the command line arguments
var Args Arguments

// isInvitationPossible returns true if an invitation to the organization is
// possible/allowed
//
// Checks for
//  - organization is not personal
//  - member count is not exceeding limits
func isInvitationPossible(organization *mono_models.Organization, numInvites int) *failures.Failure {
	// don't allow personal organizations
	if organization.Personal {
		return failures.FailUser.New(locale.T(
			"invite_personal_org_err",
		))
	}

	limits, fail := model.FetchOrganizationLimits(organization.URLname)
	if fail != nil {
		return failures.FailRuntime.New(locale.T("invite_limit_fetch_err"))
	}

	requestedMemberCount := organization.MemberCount + int64(numInvites)
	if requestedMemberCount > limits.UsersLimit {
		memberCountExceededBy := requestedMemberCount - limits.UsersLimit
		remainingFreeSeats := limits.UsersLimit - organization.MemberCount

		return failures.FailUser.New(locale.T("invite_member_limit_err", map[string]string{
			"Organization":   organization.Name,
			"UserLimit":      strconv.FormatInt(limits.UsersLimit, 10),
			"ExceededBy":     strconv.FormatInt(memberCountExceededBy, 10),
			"RemainingUsers": strconv.FormatInt(remainingFreeSeats, 10),
		}))
	}
	return nil
}

func promptOrgRole(p prompt.Prompter, emails []string, orgName string) (OrgRole, *failures.Failure) {
	choices, orgRoleNames := orgRoleChoices()
	var inviteString string
	if len(emails) == 1 {
		inviteString = emails[0]
	} else {
		inviteString = fmt.Sprintf("%d %s", len(emails), locale.T("users_plural"))
	}
	selection, fail := p.Select(locale.T("invite_select_org_role", map[string]interface{}{
		"Invitees":     inviteString,
		"Organization": orgName,
	}), choices, "")
	if fail != nil {
		return None, fail
	}
	res, ok := orgRoleNames[selection]
	if !ok {
		return None, failures.FailUserInput.New(locale.T("invite_role_needs_selection"))
	}

	return res, nil
}

func selectOrgRole(prompter prompt.Prompter, roleString string, emails []string, orgName string) (OrgRole, *failures.Failure) {
	if roleString == "" {
		return promptOrgRole(prompter, emails, orgName)
	}

	switch roleString {
	case "member":
		return Member, nil
	case "owner":
		return Owner, nil
	}
	return None, failures.FailUserInput.New("invite_invalid_role_string")
}

func sendInvite(org *mono_models.Organization, orgRole OrgRole, email string) *failures.Failure {
	// ignore the invitation for now
	_, fail := model.InviteUserToOrg(org, orgRole == Owner, email)
	if fail != nil {
		return fail
	}

	print.Line(locale.T("invite_invitation_sent", map[string]string{
		"Email": email,
	}))

	return nil
}

func callInParallel(callback func(arg string) *failures.Failure, args []string) []*failures.Failure {

	var wg sync.WaitGroup
	// never make more than 10 requests in parallel
	semaphoreChan := make(chan bool, MaxParallelRequests)
	defer close(semaphoreChan)

	errorChan := make(chan *failures.Failure, len(args))

	for _, arg := range args {
		wg.Add(1)
		semaphoreChan <- true
		go func(argRec string) {
			defer wg.Done()
			defer func() {
				<-semaphoreChan
			}()

			fail := callback(argRec)
			if fail != nil {
				errorChan <- fail
			}
		}(arg)
	}

	wg.Wait()
	close(errorChan)

	var fails []*failures.Failure
	for fail := range errorChan {
		fails = append(fails, fail)
	}
	return fails
}

func sendInvites(org *mono_models.Organization, orgRole OrgRole, emails []string) []*failures.Failure {

	fails := callInParallel(func(email string) *failures.Failure {
		return sendInvite(org, orgRole, email)
	}, emails)

	return fails
}

// Execute the organizations command.
func Execute(cmd *cobra.Command, args []string) {
	prj := project.Get()

	orgName := prj.Owner()
	if Args.Organization != "" {
		orgName = Args.Organization
	}
	emails := strings.Split(Args.EmailList, ",")
	orgRole, fail := selectOrgRole(prompter, Args.RoleString, emails, orgName)

	// Errors are handled in selectOrgRole, so we can just return if orgRole is None.
	if fail != nil {
		failures.Handle(fail, locale.T("invite_invalid_role_string"))
		return
	}

	organization, fail := model.FetchOrgByURLName(orgName)
	if fail != nil {
		failures.Handle(fail, locale.Tr("fetch_org_err", orgName))
		return
	}

	fail = isInvitationPossible(organization, len(emails))
	if fail != nil {
		// Here I am just handling an error with an error message that is already
		// tailored for the user, hence the second argument is ""
		failures.Handle(fail, "")
		return
	}

	fails := sendInvites(organization, orgRole, emails)
	if len(fails) > 0 {
		failures.Handle(fails[0], "invite_invitation_err")
		for i := 2; i < len(fails); i++ {
			print.Error(fails[i].Error())
		}
	}

	// if at least one invitation could be send, remind user to refresh secrets
	if len(fails) < len(emails) {
		print.Info(locale.T("invite_org_secrets_reminder"))
	}
}
