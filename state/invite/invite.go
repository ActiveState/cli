package invite

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
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

// checkInvite returns true if an invitation to the organization is
// possible/allowed
//
// Checks for
//  - organization is not personal
//  - member count is not exceeding limits
func checkInvites(organization *mono_models.Organization, numInvites int) *failures.Failure {
	// don't allow personal organizations
	if organization.Personal {
		return failures.FailUser.New(locale.T(
			"invite_personal_org_err",
		))
	}

	limits, fail := model.FetchOrganizationLimits(organization.Urlname)
	if fail != nil {
		return failures.FailRuntime.New(locale.T("invite_limit_fetch_err"))
	}

	requestedMemberCount := organization.MemberCount + int64(numInvites)
	if limits.UsersLimit != nil && requestedMemberCount > *limits.UsersLimit {
		memberCountExceededBy := requestedMemberCount - *limits.UsersLimit

		return failures.FailUser.New(locale.T("invite_member_limit_err", map[string]string{
			"Organization": organization.Name,
			"UserLimit":    strconv.FormatInt(*limits.UsersLimit, 10),
			"ExceededBy":   strconv.FormatInt(memberCountExceededBy, 10),
		}))
	}
	return nil
}

func promptOrgRole(p prompt.Prompter, emails []string, orgName string) OrgRole {
	choices := orgRoleChoices()
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
		return None
	}
	switch selection {
	case choices[0]:
		return Owner
	case choices[1]:
		return Member
	}
	return None
}

func selectOrgRole(prompter prompt.Prompter, roleString string, emails []string, orgName string) OrgRole {
	if roleString == "" {
		return promptOrgRole(prompter, emails, orgName)
	}

	switch roleString {
	case "member":
		return Member
	case "owner":
		return Owner
	}
	print.Error(locale.T("invite_invalid_role_string"))
	return None
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
	semaphoreChan := make(chan struct{}, MaxParallelRequests)
	errorChan := make(chan *failures.Failure, len(args))

	for _, arg := range args {
		wg.Add(1)
		semaphoreChan <- struct{}{}
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

	errCount := 0
	fails := make([]*failures.Failure, len(errorChan))
	for fail := range errorChan {
		fails[errCount] = fail
		errCount++
	}
	return fails
}

func sendInvites(org *mono_models.Organization, orgRole OrgRole, emails []string) bool {

	fails := callInParallel(func(email string) *failures.Failure {
		return sendInvite(org, orgRole, email)
	}, emails)

	for _, fail := range fails {
		failures.Handle(fail, locale.T("invite_invitation_err"))
	}
	// if at least one invite worked, send reminder to sync secrets
	numErrors := len(fails)
	if numErrors < len(emails) {
		print.Info(locale.T("invite_org_secrets_reminder"))
	}
	return numErrors == 0
}

// Execute the organizations command.
func Execute(cmd *cobra.Command, args []string) {
	prj := project.Get()

	// MD-TODO: test if we need to QueryEscape the Owner()...
	var orgName string = prj.Owner()
	if Args.Organization != "" {
		orgName = Args.Organization
	}
	emails := strings.Split(Args.EmailList, ",")
	orgRole := selectOrgRole(prompter, Args.RoleString, emails, orgName)
	// MD-TODO: I think this is the correct behavior: give the user the chance
	// to cancel the action here.  I don't think that an output will be
	// necessary here.
	if orgRole == None {
		return
	}

	organization, fail := model.FetchOrgByURLName(orgName)
	if fail != nil {
		failures.Handle(fail, locale.T("invite_fetch_org_err", map[string]string{
			"Organization": orgName,
		}))
		return
	}

	fail = checkInvites(organization, len(emails))
	if fail != nil {
		// Here I am just handling an error with an error message that is already
		// tailored for the user, hence the second argument is ""
		failures.Handle(fail, "")
		return
	}

	sendInvites(organization, orgRole, emails)
}
