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
	cancel   context.CancelFunc
}

func NewLocalProjectsUpdater(menuItem *systray.MenuItem) *LocalProjectsUpdater {
	return &LocalProjectsUpdater{menuItem, []*localProjectsMenuItem{}}
}

func (u *LocalProjectsUpdater) Update(projects []*graph.Project) {
	for _, item := range u.items {
		item.remove()
	}

	u.items = make([]*localProjectsMenuItem, 0)
	for _, project := range projects {
		u.addLocalProject(project)
	}
}

func (u *LocalProjectsUpdater) addLocalProject(project *graph.Project) {
	ctx, cancel := context.WithCancel(context.Background())

	mProject := u.menuItem.AddSubMenuItem(project.Namespace, "")
	u.items = append(u.items, &localProjectsMenuItem{mProject, project, cancel})

	if len(project.Locations) == 1 {
		go waitForClick(ctx, mProject, project.Namespace, project.Locations[0])
		return
	}

	for _, location := range project.Locations {
		mLocation := mProject.AddSubMenuItem(location, "")
		go waitForClick(ctx, mLocation, project.Namespace, location)
	}
}

func (u *localProjectsMenuItem) remove() {
	u.cancel()
	u.menuItem.Hide()
}

func waitForClick(ctx context.Context, menuItem *systray.MenuItem, namespace, location string) {
	for {
		select {
		case <-menuItem.ClickedCh:
			err := open.Prompt(fmt.Sprintf("state activate %s --path %s", namespace, location))
			if err != nil {
				logging.Error("Could not open local projects prompt for project %s, got error: %v", namespace, err)
			}
		case <-ctx.Done():
			return
		}
	}
}
