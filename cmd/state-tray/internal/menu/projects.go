package menu

import (
	"fmt"

	"github.com/ActiveState/cli/cmd/state-tray/internal/open"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/getlantern/systray"
)

type LocalProjectsUpdater struct {
	menuItem *systray.MenuItem
	items    []*localProjectsMenuItem
}

type localProjectsMenuItem struct {
	menuItem  *systray.MenuItem
	namespace string
	location  string
	close     chan struct{}
}

func NewLocalProjectsUpdater(menuItem *systray.MenuItem) *LocalProjectsUpdater {
	return &LocalProjectsUpdater{menuItem, []*localProjectsMenuItem{}}
}

func (u *LocalProjectsUpdater) Update(projects []*graph.Project) {
	u.removeItems()

	u.items = []*localProjectsMenuItem{}
	for _, project := range projects {
		for _, location := range project.Locations {
			mProject := u.menuItem.AddSubMenuItem(project.Namespace, location)
			u.items = append(u.items, &localProjectsMenuItem{mProject, project.Namespace, location, make(chan struct{})})
		}
	}

	u.startEventLoops()
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
			err := open.Prompt(fmt.Sprintf("state activate %s --path %s", i.namespace, i.location))
			if err != nil {
				logging.Error("Could not open local projects prompt for project %s, got error: %v", i.namespace, err)
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
