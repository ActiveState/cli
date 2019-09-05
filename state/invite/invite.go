package invite

import (
	"errors"
	"regexp"
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
	"github.com/thoas/go-funk"
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
			Description: "invite members into this organization.  Default value is the organization of the current project",
			Type:        commands.TypeString,
			StringVar:   &Args.Organization,
		},
	},

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "<email1>,[<email2>,..]",
			Description: "arg_state_invite_emails",
			Required:    true,
			Variable:    &Args.EmailList,
			Validator:   emailListValidator,
		},
	},
}

var prompter prompt.Prompter

func init() {
	prompter = prompt.New()
}

func emailListValidator(_ *commands.Argument, value string) error {
	emailRe := regexp.MustCompile(
		"^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?" +
			"(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$",
	)

	candidates := strings.Split(value, ",")
	rejectedEmails := funk.FilterString(candidates, func(candidate string) bool {
		return !emailRe.MatchString(candidate)
	})
	if len(rejectedEmails) > 0 {
		return errors.New(locale.Tr("invite_invalid_email_args", strings.Join(rejectedEmails, ",")))
	}
	return nil
}

// Args stores command line argument values for the invite command
var Args struct {
	EmailList    string
	Organization string
}

// checkInvite returns true if an invitation to the organization is
// possible/allowed
//
// Checks for
//  - organization is not personal
//  - member count is not exceeding limits
//
// Note: I would prefer to return an error strings here for easier testing, but
// failures.Handle() needs to be called and already prints its output.  This
// could maybe be improved in the future...?
func checkInvites(organization *mono_models.Organization, numInvites int) bool {
	// ignore personal organizations
	if organization.Personal {
		print.Error(locale.T("invite_personal_org_err"))
		return false
	}

	limits, fail := model.FetchOrganizationLimits(organization.Urlname)
	if fail != nil {
		failures.Handle(fail, locale.T("invite_limit_fetch_err"))
		return false
	}

	requestedMemberCount := organization.MemberCount + int64(numInvites)
	if limits.UsersLimit != nil && requestedMemberCount > *limits.UsersLimit {
		memberCountExceededBy := requestedMemberCount - *limits.UsersLimit

		print.Error(locale.T("invite_member_limit_err", memberCountExceededBy, *limits.UsersLimit))
		return false
	}
	return true
}

// OrgRole is an enumeration of the roles that user can have in an organization
type OrgRole int

const (
	// None means no role selected
	None OrgRole = iota
	// Owner of an organization
	Owner
	// Member in an organization
	Member
)

var orgRoleChoices []string

func init() {
	orgRoleChoices = []string{
		locale.T("org_role_choice_owner"),
		locale.T("org_role_choice_member"),
	}
}

func selectOrgRole(p prompt.Prompter, numInvites int) OrgRole {
	selection, fail := p.Select(locale.T("invite_select_org_role", numInvites), orgRoleChoices[:], "")
	if fail != nil {
		return None
	}
	switch selection {
	case orgRoleChoices[0]:
		return Owner
	case orgRoleChoices[1]:
		return Member
	}
	return None
}

func sendInvite(org *mono_models.Organization, orgRole OrgRole, email string) *failures.Failure {
	// ignore the invitation for now
	_, fail := model.InviteUserToOrg(org, orgRole == Owner, email)
	if fail != nil {
		return fail
	}

	return nil
}

func callInParallel(callback func(arg string) *failures.Failure, args []string) []*failures.Failure {

	var wg sync.WaitGroup
	// never make more than 10 requests in parallel
	semaphoreChan := make(chan struct{}, MaxParallelRequests)
	errorChan := make(chan *failures.Failure, len(args))

	for _, arg := range args {
		wg.Add(1)
		go func(argRec string) {
			defer wg.Done()
			semaphoreChan <- struct{}{}
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

	numErrors := 0
	for fail := range errorChan {
		failures.Handle(fail, locale.T("invite_org_err"))
		numErrors++
	}
}

func sendInvites(org *mono_models.Organization, orgRole OrgRole, emails []string) bool {
	var wg sync.WaitGroup
	// never make more than 10 requests in parallel
	semaphoreChan := make(chan struct{}, MaxParallelInvites)
	errorChan := make(chan *failures.Failure, len(emails))

	for _, email := range emails {
		wg.Add(1)
		go func(invitee string) {
			defer wg.Done()
			semaphoreChan <- struct{}{}
			defer func() {
				<-semaphoreChan
			}()

			fail := sendInvite(org, orgRole, invitee)
			if fail != nil {
				errorChan <- fail
			}
			print.Info(locale.T("invite_org_sent", invitee))
		}(email)
	}

	wg.Wait()
	close(errorChan)

	numErrors := 0
	for fail := range errorChan {
		failures.Handle(fail, locale.T("invite_org_err"))
		numErrors++
	}
	// if at least one invite worked, send reminder to sync secrets
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
	organization, fail := model.FetchOrgByURLName(orgName)
	if fail != nil {
		failures.Handle(fail, locale.T("invite_org_err"))
		return
	}

	emails := strings.Split(Args.EmailList, ",")

	if !checkInvites(organization, len(emails)) {
		return
	}

	orgRole := selectOrgRole(prompter, len(emails))

	// MD-TODO: I think this is the correct behavior: give the user the chance
	// to cancel the action here.  I don't think that an output will be
	// necessary here.
	if orgRole == None {
		return
	}

	sendInvites(organization, orgRole, emails)
}
