package events

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "events",
	Description: "events_description",
	Run:         Execute,
}

// Execute List configured eventss
// If no events trigger name given, lists all
// Otherwise shows configured eventss for given events trigger
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Events")
	var T = locale.T

	pj := project.Get()

	print.Info(T("events_listing"))
	print.Line("")

	rows := [][]interface{}{}
	for _, event := range pj.Events() {
		rows = append(rows, []interface{}{event.Name(), event.Value()})
	}

	if len(rows) == 0 {
		print.Line(locale.T("events_empty"))
		return
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{T("events_header_event"), T("events_header_value")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))
}
