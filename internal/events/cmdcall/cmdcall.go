package cmdcall

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Configurer
}

// CmdCall manages dependencies for the handling of events triggered by command
// calls.
type CmdCall struct {
	out       output.Outputer
	proj      *project.Project
	subshell  subshell.SubShell
	cmdList   string
	p         primeable
	scriptrun *scriptrun.ScriptRun
}

// New returns a prepared pointer to an instance of CmdCall.
func New(p primeable, cmdList string) *CmdCall {
	return &CmdCall{
		out:       p.Output(),
		proj:      p.Project(),
		subshell:  p.Subshell(),
		cmdList:   cmdList,
		p:         p,
		scriptrun: scriptrun.New(p.Auth(), p.Output(), p.Subshell(), p.Project(), p.Config()),
	}
}

// Run executes the event handling logic by running any relevant scripts.
func (cc *CmdCall) Run(eventType project.EventType) error {
	logging.Debug("cmdcall")

	if cc.proj == nil {
		return nil
	}

	var events []*project.Event
	for _, event := range cc.proj.Events() {
		if event.Name() != string(eventType) {
			continue
		}

		scopes, err := event.Scope()
		if err != nil {
			return locale.WrapError(
				err, "cmdcall_event_err_get_scope",
				"Cannot obtain scopes for event '[NOTICE]{{.V0}}[/RESET]'",
				event.Name(),
			)
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
			return locale.WrapError(
				err, "cmdcall_event_err_get_value",
				"Cannot get triggered event value for event '[NOTICE]{{.V0}}[/RESET]'",
				event.Name(),
			)
		}

		ss := strings.Split(val, " ")
		if len(ss) == 0 {
			return locale.NewError(
				"cmdcall_event_err_get_value",
				"Triggered event value is empty for event '[NOTICE]{{.V0}}[/RESET]'",
				event.Name(),
			)
		}

		scriptName, scriptArgs := ss[0], ss[1:]
		if err := cc.scriptrun.Run(cc.proj.ScriptByName(scriptName), scriptArgs); err != nil {
			return locale.WrapError(
				err, "cmdcall_event_err_script_run",
				"Failure running defined script '[NOTICE]{{.V0}}[/RESET]' for event '[NOTICE]{{.V1}}[/RESET]'",
				scriptName, event.Name(),
			)
		}
	}

	return nil
}
