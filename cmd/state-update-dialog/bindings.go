package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/cmd/state-update-dialog/internal/lockedprj"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/rollbar/rollbar-go"
)

type Bindings struct {
	update         *updater.AvailableUpdate
	lockedProjects map[string][]lockedprj.LockedCheckout
	changelog      string
}

func (b *Bindings) CurrentVersion() string {
	return constants.VersionNumber
}

func (b *Bindings) AvailableVersion() string {
	return b.update.Version
}

func (b *Bindings) Warning() string {
	if len(b.lockedProjects) == 0 {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("The following local projects will be affected if the latest update to State Tool is applied:<ul>")
	for name, prjs := range b.lockedProjects {
		buf.WriteString(fmt.Sprintf("<li><span>%s</span>: <ul>", name))
		for _, prj := range prjs {
			buf.WriteString(fmt.Sprintf("<li>The activestate.yaml file at <code>%s</code> is locked at State Tool version <em>%s@%s</em> for the above project, the latest update can impact the project adversely. Please run <code>state update lock --force</code> after updating.</li>", prj.Path, prj.Channel, prj.Version))
		}
		buf.WriteString("</ul></li>")
	}

	buf.WriteString("</ul>")

	return buf.String()
}

func (b *Bindings) Changelog() string {
	return b.changelog
}

func (b *Bindings) Exit() {
	// This is SUPER dirty, but as of right now wails leaves us no other choice:
	// https://github.com/wailsapp/wails/issues/693
	events.WaitForEvents(1*time.Second, rollbar.Close)
	os.Exit(0)
}

func (b *Bindings) DebugMode() bool {
	args := strings.Join(os.Args, "")
	return strings.Contains(args, "wails.BuildMode=debug") || strings.Contains(args, "go-build")
}
