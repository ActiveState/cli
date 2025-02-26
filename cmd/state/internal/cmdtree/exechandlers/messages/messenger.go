package messages

import (
	"context"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	msgs "github.com/ActiveState/cli/internal/messages"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Messenger struct {
	out      output.Outputer
	svcModel *model.SvcModel
}

func New(out output.Outputer, svcModel *model.SvcModel) *Messenger {
	return &Messenger{
		out:      out,
		svcModel: svcModel,
	}
}

func (m *Messenger) OnExecStart(_ *captain.Command, _ []string) error {
	logging.Debug("Checking for messages")
	if m.out.Type().IsStructured() {
		return nil
	}

	messages, err := m.svcModel.CheckMessages(context.Background())
	if err != nil {
		return errs.Wrap(err, "Could not get messages")
	}
	logging.Debug("Found %d messages", len(messages))

	for _, message := range messages {
		m.out.Notice("") // Line break before

		segments := strings.Split(message.Topic, ".")
		if len(segments) > 0 {
			switch segments[0] {
			case msgs.TopicError:
				m.handleErrorMessages(message)
			case msgs.TopicInfo:
				logging.Info("State Service reported an info message: %s", message.Message)
				m.out.Notice(message.Message)
			default:
				logging.Debug("State Service reported an unknown message: %s", message.Topic)
				m.out.Notice(message.Message) // fallback to notice for unknown types
			}
		}

		m.out.Notice("") // Line break after
	}

	return nil
}

func (m *Messenger) handleErrorMessages(message *graph.Message) {
	logging.Error("State Service reported a %s error: %s", message.Topic, message.Message)
	err := locale.NewError("err_svc_message", "[WARNING]Warning:[/RESET] {{.V0}}", message.Message)
	m.out.Error(err)
}
