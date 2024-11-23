package messages

import (
	"context"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
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
	if m.out.Type().IsStructured() {
		// No point showing messages on structured output (eg. json)
		return nil
	}

	messages, err := m.svcModel.CheckMessages(context.Background())
	if err != nil {
		return errs.Wrap(err, "Could not get messages")
	}

	for _, message := range messages {
		m.out.Notice("") // Line break before
		// TODO: Add handling for different message types
		m.out.Notice(message.Message)
	}

	return nil
}
