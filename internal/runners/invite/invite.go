package invite

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
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
}

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Prompter
}

func New(prime primeable) *invite {
	return &invite{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
	}
}

func (i *invite) Run(params *Params) error {
	if i.project == nil {
		return locale.NewInputError("err_no_projectfile", "Must be in a project directory.")
	}

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

	emails := strings.Split(params.EmailList, ",")

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

type outputFormat struct {
	Email string `locale:"email,Email"`
	Error error  `locale:"error,Error"`
}

func (f *outputFormat) MarshalOutput(format output.Format) interface{} {
	switch format {
	case output.PlainFormatName:
		success := locale.Tl("ok", "Ok")
		if f.Error != nil {
			success = locale.Tl("err_invite", "Failed: {{.V0}}", f.Error.Error())
		}
		return locale.Tl("invite_success", "Sending to {{.V0}} ... {{.V1}}", f.Email, success)
	}

	return f
}

func (i *invite) sendSingle(orgName string, asOwner bool, email string) error {
	// ignore the invitation for now
	_, err := model.InviteUserToOrg(orgName, asOwner, email)
	if err != nil {
		i.out.Error(&outputFormat{email, err})
		return err
	}

	i.out.Print(&outputFormat{email, nil})

	return nil
}
