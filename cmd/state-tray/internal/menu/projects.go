package menu

import (
	"context"
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
	menuItem *systray.MenuItem
	project  *graph.Project
	done     <-chan struct{}
	cancel   context.CancelFunc
}

func NewLocalProjectsUpdater(menuItem *systray.MenuItem) *LocalProjectsUpdater {
	return &LocalProjectsUpdater{menuItem, []*localProjectsMenuItem{}}
}

func (u *LocalProjectsUpdater) Update(projects []*graph.Project) {
	for _, item := range u.items {
		item.remove()
	}

	u.items = make([]*localProjectsMenuItem, len(projects))
	for i, project := range projects {
		item := newLocalProjectMenuItem(project, u.menuItem)
		u.items[i] = item
		item.startEventLoop()
	}
}

func newLocalProjectMenuItem(project *graph.Project, menuItem *systray.MenuItem) *localProjectsMenuItem {
	ctx, cancel := context.WithCancel(context.Background())

	mProject := menuItem.AddSubMenuItem(project.Namespace, "")
	return &localProjectsMenuItem{mProject, project, ctx.Done(), cancel}
}

func (i *localProjectsMenuItem) startEventLoop() {
	if len(i.project.Locations) == 1 {
		go i.eventLoop(i.project.Locations[0])
		return
	}

	for _, location := range i.project.Locations {
		mLocation := i.menuItem.AddSubMenuItem(location, "")
		subItem := i.newSubMenuItem(mLocation)
		go subItem.eventLoop(location)
	}
}

func (i *localProjectsMenuItem) newSubMenuItem(menuItem *systray.MenuItem) *localProjectsMenuItem {
	return &localProjectsMenuItem{menuItem, i.project, i.done, i.cancel}
}

func (i *localProjectsMenuItem) eventLoop(location string) {
	for {
		select {
		case <-i.menuItem.ClickedCh:
			err := open.Prompt(fmt.Sprintf("state activate %s --path %s", i.project.Namespace, location))
			if err != nil {
				logging.Error("Could not open local projects prompt for project %s, got error: %v", i.project.Namespace, err)
			}
		case <-i.done:
			return
		}
	}
}

func (i *localProjectsMenuItem) remove() {
	i.cancel()
	i.menuItem.Hide()
}
