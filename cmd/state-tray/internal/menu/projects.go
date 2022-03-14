package menu

import (
	"fmt"
	"unicode/utf8"

	"github.com/ActiveState/cli/cmd/state-tray/internal/open"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/getlantern/systray"
)

type LocalProjectsUpdater struct {
	menuItem *systray.MenuItem
	items    []*localProjectsMenuItem
}

type localProjectsMenuItem struct {
	menuItem       *systray.MenuItem
	namespace      string
	location       string
	customCallback func()
	close          chan struct{}
}

const (
	maxEntryLength = 30
	ellipsis       = "\u2026"
)

func NewLocalProjectsUpdater(menuItem *systray.MenuItem) *LocalProjectsUpdater {
	return &LocalProjectsUpdater{menuItem, []*localProjectsMenuItem{}}
}

func (u *LocalProjectsUpdater) Update(projects []*graph.Project) {
	u.removeItems()

	u.items = []*localProjectsMenuItem{}
	for _, project := range projects {
		for _, location := range project.Locations {
			mProject := u.menuItem.AddSubMenuItem(trimEntry(location), location)
			u.items = append(u.items, &localProjectsMenuItem{mProject, project.Namespace, location, nil, make(chan struct{})})
		}
	}

	if len(projects) == 0 {
		mitem := u.menuItem.AddSubMenuItem("Get Started", "")
		cb := func() {
			open.Browser(constants.DocumentationURLGetStarted)
		}
		u.items = append(u.items, &localProjectsMenuItem{mitem, "", "", cb, make(chan struct{})})
	}

	u.startEventLoops()
}

func trimEntry(entry string) string {
	entryLength := utf8.RuneCountInString(entry)
	if entryLength <= maxEntryLength {
		return entry
	}

	e := []rune(entry)
	return fmt.Sprintf("%s%s", ellipsis, string(e[entryLength-maxEntryLength:]))
}

func (u *LocalProjectsUpdater) removeItems() {
	for _, item := range u.items {
		item.remove()
	}
}

func (u *LocalProjectsUpdater) startEventLoops() {
	for _, item := range u.items {
		go item.eventLoop()
	}
}

func (i *localProjectsMenuItem) eventLoop() {
	for {
		select {
		case <-i.menuItem.ClickedCh:
			if i.customCallback != nil {
				i.customCallback()
			} else {
				cmd := fmt.Sprintf("%s activate %s --path %s", appinfo.StateApp().Exec(), i.namespace, i.location)
				ns, err := project.ParseNamespace(i.namespace)
				if err != nil || !ns.IsValid() {
					cmd = fmt.Sprintf("%s activate --path %s", appinfo.StateApp().Exec(), i.location)
				}
				err = open.TerminalAndWait(cmd)
				if err != nil {
					multilog.Error("Could not open local projects prompt for project %s, got error: %v", i.namespace, err)
				}
			}
		case <-i.close:
			return
		}
	}
}

func (i *localProjectsMenuItem) remove() {
	i.close <- struct{}{}
	i.menuItem.Hide()
}
