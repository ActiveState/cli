package notifier

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/model"
	"golang.org/x/net/context"
)

type Notifier struct {
	out           output.Outputer
	svcModel      *model.SvcModel
	notifications []*graph.NotificationInfo
}

func New(out output.Outputer, svcModel *model.SvcModel) *Notifier {
	return &Notifier{
		out:      out,
		svcModel: svcModel,
	}
}

func (m *Notifier) OnExecStart(cmd *captain.Command, _ []string) error {
	if m.out.Type().IsStructured() {
		// No point showing notifications on non-plain output (eg. json)
		return nil
	}

	if cmd.Name() == "update" {
		return nil // do not print update/deprecation warnings/notifications when running `state update`
	}

	cmds := cmd.JoinedCommandNames()
	flags := cmd.ActiveFlagNames()

	notifications, err := m.svcModel.CheckNotifications(context.Background(), cmds, flags)
	if err != nil {
		multilog.Error("Could not report notifications as CheckNotifications return an error: %s", errs.JoinMessage(err))
	}

	m.notifications = notifications

	logging.Debug("Received %d notifications to print", len(notifications))

	if err := m.PrintByPlacement(graph.NotificationPlacementTypeBeforeCmd); err != nil {
		return errs.Wrap(err, "notification error occurred before cmd")
	}

	return nil
}

func (m *Notifier) OnExecStop(cmd *captain.Command, _ []string) error {
	if m.out.Type().IsStructured() {
		// No point showing notification on non-plain output (eg. json)
		return nil
	}

	if cmd.Name() == "update" {
		return nil // do not print update/deprecation warnings/notifications when running `state update`
	}

	if err := m.PrintByPlacement(graph.NotificationPlacementTypeAfterCmd); err != nil {
		return errs.Wrap(err, "notification error occurred before cmd")
	}

	return nil
}

func (m *Notifier) PrintByPlacement(placement graph.NotificationPlacementType) error {
	exit := []string{}

	notifications := []*graph.NotificationInfo{}
	for _, notification := range m.notifications {
		if notification.Placement != placement {
			logging.Debug("Skipping notification %s as it's placement (%s) does not match %s", notification.ID, notification.Placement, placement)
			notifications = append(notifications, notification)
			continue
		}

		if placement == graph.NotificationPlacementTypeAfterCmd {
			m.out.Notice("") // Line break after
		}

		logging.Debug("Printing notification: %s, %s", notification.ID, notification.Message)
		m.out.Notice(notification.Message)

		if placement == graph.NotificationPlacementTypeBeforeCmd {
			m.out.Notice("") // Line break before
		}

		if notification.Interrupt == graph.NotificationInterruptTypePrompt {
			if m.out.Config().Interactive {
				m.out.Print(locale.Tl("notifier_prompt_continue", "Press ENTER to continue."))
				fmt.Scanln(ptr.To("")) // Wait for input from user
			} else {
				logging.Debug("Skipping notification prompt as we're not in interactive mode")
			}
		}

		if notification.Interrupt == graph.NotificationInterruptTypeExit {
			exit = append(exit, notification.ID)
		}
	}

	m.notifications = notifications

	if len(exit) > 0 {
		// It's the responsibility of the notification to give the user context as to why this exit happened.
		// We pass an input error here to ensure this doesn't get logged.
		return errs.Silence(errs.WrapExitCode(errs.New("Following notifications triggered exit: %s", strings.Join(exit, ", ")), 1))
	}

	return nil
}
