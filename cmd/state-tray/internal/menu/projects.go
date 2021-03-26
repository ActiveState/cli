package menu

import (
	"context"
	"fmt"

	"github.com/ActiveState/cli/cmd/state-tray/internal/open"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/getlantern/systray"
)

type LocalProjectsMenu struct {
	*systray.MenuItem
	items []localProjectMenuItem
}

type localProjectMenuItem struct {
	*systray.MenuItem
	project *graph.Project
	cancel  context.CancelFunc
}

func NewLocalProjectsMenu(menuItem *systray.MenuItem) *LocalProjectsMenu {
	return &LocalProjectsMenu{menuItem, []localProjectMenuItem{}}
}

func (m *LocalProjectsMenu) Populate(projects []*graph.Project) {
	for _, project := range projects {
		m.addLocalProject(project)
	}
}

func (m *LocalProjectsMenu) Reload(projects []*graph.Project) {
	for _, item := range m.items {
		item.remove()
	}
	m.items = make([]localProjectMenuItem, 0)
	m.Populate(projects)
}

func (m *LocalProjectsMenu) addLocalProject(project *graph.Project) {
	ctx, cancel := context.WithCancel(context.Background())

	mProject := m.AddSubMenuItem(project.Namespace, "")
	m.items = append(m.items, localProjectMenuItem{mProject, project, cancel})

	if len(project.Locations) == 1 {
		go waitForClick(ctx, mProject, project.Namespace, project.Locations[0])
		return
	}

	for _, location := range project.Locations {
		mLocation := mProject.AddSubMenuItem(location, "")
		go waitForClick(ctx, mLocation, project.Namespace, location)
	}
}

func (m *localProjectMenuItem) remove() {
	m.cancel()
	m.Hide()
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
