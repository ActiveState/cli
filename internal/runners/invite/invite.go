package invite

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// MaxParallelRequests is the maximum number of invite requests that we want to send in parallel
const MaxParallelRequests = 10

type Params struct {
	Org       Org
	Role      Role
	EmailList string
}

type invite struct {
	project *project.Project
	out     output.Outputer
	prompt  prompt.Prompter
	auth    *authentication.Auth
}

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Prompter
	primer.Auther
}

func New(prime primeable) *invite {
	return &invite{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
		prime.Auth(),
	}
}

func (i *invite) Run(params *Params, args []string) error {
	if i.project == nil {
		return locale.NewInputError("err_no_projectfile", "Must be in a project directory.")
	}
	if !i.auth.Authenticated() {
		return locale.NewInputError("err_invite_not_logged_in", "You need to authenticate with '[ACTIONABLE]state auth[/RESET]' before you can invite new members.")
	}

	if len(args) > 1 {
		params.EmailList = strings.Join(args, ",")
	} // otherwise CSV-separated list of e-mails is already in params.EmailList

	org := params.Org
	if org.String() == "" {
		if err := (&org).Set(i.project.Owner()); err != nil {
			return locale.WrapInputError(err, "err_invite_org_current", "Could not use the owner of your current project.")
		}
	}

	role := params.Role
	if role == Unknown {
		var err error
		if role, err = i.promptForRole(); err != nil {
			return err
		}
	}

	multipleCommas := regexp.MustCompile(",,+")
	emailList := strings.Trim(multipleCommas.ReplaceAllString(params.EmailList, ","), ",")
	emails := strings.Split(emailList, ",")

	if err := org.CanInvite(len(emails)); err != nil {
		return locale.WrapError(err, "err_caninvite", "Cannot invite users to {{.V0}}.", org.String())
	}

	sent, err := i.send(org.String(), role == Owner, emails)
	if sent > 0 {
		i.out.Notice(locale.Tl(
			"invite_org_secrets_reminder",
			"\n{{.V0}} out of {{.V1}} invites were sent.\nYou should run 'state secrets sync' once invitees have accepted their invitation and authenticated with the State Tool.",
			strconv.Itoa(sent), strconv.Itoa(len(emails)),
		))
	}

	if err != nil {
		if sent > 0 {
			return locale.WrapError(err, "err_invite_send", "\nNot all invites were able to send.")
		} else {
			return locale.WrapError(err, "err_invite_send", "\nCould not send invites.")
		}
	}

	return nil
}

func (i *invite) promptForRole() (Role, error) {
	choices := roleNames()
	selection, err := i.prompt.Select(locale.Tl("invite_role", "Role"), locale.Tl("invite_select_org_role", "What role should the user(s) be given?"), choices, new(string))
	if err != nil {
		return -1, err
	}
	var role Role
	if err := (&role).Set(selection); err != nil {
		return role, err
	}
	return role, nil
}

func (i *invite) send(orgName string, asOwner bool, emails []string) (int, error) {
	var wg sync.WaitGroup
	// never make more than 10 requests in parallel
	semaphoreChan := make(chan bool, MaxParallelRequests)
	defer close(semaphoreChan)

	errorChan := make(chan error, len(emails))

	for _, email := range emails {
		wg.Add(1)
		semaphoreChan <- true
		go func(curEmail string) {
			defer wg.Done()
			defer func() {
				<-semaphoreChan
			}()

			err := i.sendSingle(orgName, asOwner, curEmail)
			if err != nil {
				errorChan <- err
			}
		}(email)
	}

	wg.Wait()
	close(errorChan)

	errLen := len(errorChan)

	var rerr error
	for err := range errorChan {
		if rerr == nil {
			rerr = err
		} else {
			rerr = fmt.Errorf("%s\n%w", err.Error(), rerr)
		}
	}
	return len(emails) - errLen, rerr
}

func (i *invite) sendSingle(orgName string, asOwner bool, email string) error {
	// ignore the invitation for now
	_, err := model.InviteUserToOrg(orgName, asOwner, email)
	if err != nil {
		return locale.WrapError(err, "err_invite", "Failed to send invite to {{.V0}}", email)
	}

	i.out.Notice(locale.Tl("invite_success", "Sent invite to {{.V0}}"))

	return nil
}
