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
	done      chan struct{}
	children  []*localProjectsMenuItem
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
	done := make(chan struct{})
	mProject := menuItem.AddSubMenuItem(project.Namespace, "")
	item := &localProjectsMenuItem{
		menuItem:  mProject,
		namespace: project.Namespace,
		done:      done,
		children:  []*localProjectsMenuItem{},
	}

	for i, location := range project.Locations {
		if i == 0 && i == len(project.Locations)-1 {
			item.location = location
			break
		}

		mLocation := mProject.AddSubMenuItem(location, "")
		item.addChild(mLocation, location)
	}

	return item
}

func (i *localProjectsMenuItem) addChild(menuItem *systray.MenuItem, location string) {
	i.children = append(i.children, &localProjectsMenuItem{
		menuItem:  menuItem,
		namespace: i.namespace,
		location:  location,
		done:      i.done,
	})
}

func (i *localProjectsMenuItem) startEventLoop() {
	if len(i.children) == 0 {
		go i.eventLoop()
	}

	for _, child := range i.children {
		go child.eventLoop()
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
		case <-i.done:
			return
		}
	}
}

func (i *localProjectsMenuItem) remove() {
	i.done <- struct{}{}
	i.menuItem.Hide()
}
