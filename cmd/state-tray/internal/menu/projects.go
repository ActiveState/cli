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
	// TODO: This should be a project.project
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
	for _, project := range projects {
		// TODO: Need to address the case of a new checkout
		if m.containsProject(project) {
			continue
		}
		m.addLocalProject(project)
	}
}

func (m *LocalProjectsMenu) containsProject(project *graph.Project) bool {
	for _, item := range m.items {
		if item.project.Owner == project.Owner && item.project.Name == project.Name {
			return true
		}
	}
	return false
}

func (m *LocalProjectsMenu) addLocalProject(project *graph.Project) {
	ctx, cancel := context.WithCancel(context.Background())

	mProject := m.AddSubMenuItem(fmt.Sprintf("%s/%s", project.Owner, project.Name), "")
	m.items = append(m.items, localProjectMenuItem{mProject, project, cancel})

	go func(ctx context.Context, proj *graph.Project) {
		for {
			select {
			case <-mProject.ClickedCh:
				err := open.Prompt(fmt.Sprintf("state activate %s/%s --path %s", proj.Owner, proj.Name, proj.Locations[0]))
				if err != nil {
					logging.Error("Could not open local projects prompt for project %s/%s, got error: %v", proj.Owner, proj.Name, err)
				}
			case <-ctx.Done():
				mProject.Hide()
				return
			}
		}
	}(ctx, project)
}
