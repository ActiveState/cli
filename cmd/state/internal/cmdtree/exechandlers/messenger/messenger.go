package messenger

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

type Messenger struct {
	out      output.Outputer
	svcModel *model.SvcModel
	messages []*graph.MessageInfo
}

func New(out output.Outputer, svcModel *model.SvcModel) *Messenger {
	return &Messenger{
		out:      out,
		svcModel: svcModel,
	}
}

func (m *Messenger) OnExecStart(cmd *captain.Command, _ []string) error {
	if m.out.Type().IsStructured() {
		// No point showing messaging on non-plain output (eg. json)
		return nil
	}

	if cmd.Name() == "update" {
		return nil // do not print update/deprecation warnings/messages when running `state update`
	}

	cmds := cmd.JoinedCommandNames()
	flags := cmd.ActiveFlagNames()

	messages, err := m.svcModel.CheckMessages(context.Background(), cmds, flags)
	if err != nil {
		multilog.Error("Could not report messages as CheckMessages return an error: %s", errs.JoinMessage(err))
	}

	m.messages = messages

	logging.Debug("Received %d messages to print", len(messages))

	if err := m.PrintByPlacement(graph.MessagePlacementTypeBeforeCmd); err != nil {
		return errs.Wrap(err, "message error occurred before cmd")
	}

	return nil
}

func (m *Messenger) OnExecStop(cmd *captain.Command, _ []string) error {
	if m.out.Type().IsStructured() {
		// No point showing messaging on non-plain output (eg. json)
		return nil
	}

	if cmd.Name() == "update" {
		return nil // do not print update/deprecation warnings/messages when running `state update`
	}

	if err := m.PrintByPlacement(graph.MessagePlacementTypeAfterCmd); err != nil {
		return errs.Wrap(err, "message error occurred before cmd")
	}

	return nil
}

func (m *Messenger) PrintByPlacement(placement graph.MessagePlacementType) error {
	exit := []string{}

	messages := []*graph.MessageInfo{}
	for _, message := range m.messages {
		if message.Placement != placement {
			logging.Debug("Skipping message %s as it's placement (%s) does not match %s", message.ID, message.Placement, placement)
			messages = append(messages, message)
			continue
		}

		if placement == graph.MessagePlacementTypeAfterCmd {
			m.out.Notice("") // Line break after
		}

		logging.Debug("Printing message: %s", message.ID)
		m.out.Notice(message.Message)

		if placement == graph.MessagePlacementTypeBeforeCmd {
			m.out.Notice("") // Line break before
		}

		if message.Interrupt == graph.MessageInterruptTypePrompt {
			if m.out.Config().Interactive {
				m.out.Print(locale.Tl("messenger_prompt_continue", "Press ENTER to continue."))
				fmt.Scanln(ptr.To("")) // Wait for input from user
			} else {
				logging.Debug("Skipping message prompt as we're not in interactive mode")
			}
		}

		if message.Interrupt == graph.MessageInterruptTypeExit {
			exit = append(exit, message.ID)
		}
	}

	m.messages = messages

	if len(exit) > 0 {
		// It's the responsibility of the message to give the user context as to why this exit happened.
		// We pass an input error here to ensure this doesn't get logged.
		return errs.Silence(errs.WrapExitCode(errs.New("Following messages triggered exit: %s", strings.Join(exit, ", ")), 1))
	}

	return nil
}
