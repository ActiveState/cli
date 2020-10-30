package cmdcall

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Subsheller
}

// CmdCall manages dependencies for the handling of events triggered by command
// calls.
type CmdCall struct {
	out      output.Outputer
	proj     *project.Project
	subshell subshell.SubShell
	cmdList  string
	p        primeable
}

// New returns a prepared pointer to an instance of CmdCall.
func New(p primeable, cmdList string) *CmdCall {
	return &CmdCall{
		out:      p.Output(),
		proj:     p.Project(),
		subshell: p.Subshell(),
		cmdList:  cmdList,
		p:        p,
	}
}

// Run executes the event handling logic by running any relevant scripts.
func (cc *CmdCall) Run(t project.EventType) error {
	logging.Debug("cmdcall")

	if cc.proj == nil {
		return nil
	}

	var events []*project.Event
	for _, event := range cc.proj.Events() {
		if event.Name() != string(t) {
			continue
		}

		scopes, err := event.Scope()
		if err != nil {
			return err // TODO: this
		}

		for _, scope := range scopes {
			if scope == cc.cmdList {
				events = append(events, event)
			}
		}

	}

	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		val, err := event.Value()
		if err != nil {
			return err // TODO: this
		}

		ss := strings.Split(val, " ")
		if len(ss) == 0 {
			return errors.New("no script defined") // TODO: this
		}

		if !scriptrun.ProjectHasScript(cc.proj, ss[0]) {
			continue
		}

		if err := scriptrun.RunScript(cc.out, cc.subshell, cc.proj, ss[0], ss[1:]); err != nil {
			return err // TODO: this
		}
	}

	return nil
}
