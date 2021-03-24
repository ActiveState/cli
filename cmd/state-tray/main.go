package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ActiveState/cli/cmd/state-tray/internal/open"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/getlantern/systray"
	"github.com/gobuffalo/packr"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	err := run()
	if err != nil {
		logging.Error("Systray encountered an error: %v", errs.Join(err, ": "))
		os.Exit(1)
	}
}

func run() error {
	logging.CurrentHandler().SetVerbose(true)

	config, err := config.Get()
	if err != nil {
		return errs.Wrap(err, "Could not get new config instance")
	}

	model, err := model.NewSvcModel(context.Background(), config)
	if err != nil {
		return errs.Wrap(err, "Could not create new service model")
	}

	box := packr.NewBox("assets")
	systray.SetIcon(box.Bytes("icon.ico"))
	systray.SetTooltip(locale.Tl("tray_tooltip", "ActiveState State Tool"))

	mAbout := systray.AddMenuItem(
		locale.Tl("tray_about_title", "About State Tool"),
		locale.Tl("tray_about_tooltip", "Information about the State Tool"),
	)

	systray.AddSeparator()

	mDoc := systray.AddMenuItem(
		locale.Tl("tray_documentation_title", "Documentation"),
		locale.Tl("tray_documentation_tooltip", "Open State Tool Docs"),
	)

	mPlatform := systray.AddMenuItem(locale.Tl("tray_platform_title", "ActiveState Platform"), "")
	mLearn := mPlatform.AddSubMenuItem(
		locale.Tl("tray_learn_title", "Learn"),
		locale.Tl("tray_learn_tooltip", "Open ActiveState blog"),
	)
	mSupport := mPlatform.AddSubMenuItem(
		locale.Tl("tray_support_title", "Support"),
		locale.Tl("tray_support_tooltip", "Open support page"),
	)
	mAccount := mPlatform.AddSubMenuItem(
		locale.Tl("tray_account_title", "Account"),
		locale.Tl("tray_account_tooltip", "Open your account page"),
	)

	systray.AddSeparator()

	mProjects := systray.AddMenuItem(locale.Tl("tray_projects_title", "Local Projects"), "")
	cancel, err := refreshProjects(model, mProjects)
	if err != nil {
		logging.Error("Could not refresh projects, got err: %v", err)
	}

	systray.AddSeparator()

	mQuit := systray.AddMenuItem(locale.Tl("tray_exit", "Exit"), "")

	for {
		select {
		case <-mAbout.ClickedCh:
			logging.Debug("About event")
			// version, err := model.StateVersion()
			// if err != nil {
			// 	logging.Error("Could not get state version, got error: %v", err)
			// }
			// fmt.Println("Version in tray: ", version.State)
			err = open.Prompt("state --version")
			if err != nil {
				logging.Error("Could not open command prompt, got error: %v", err)
			}
		case <-mDoc.ClickedCh:
			logging.Debug("Documentation event")
			// Not implemented
		case <-mLearn.ClickedCh:
			logging.Debug("Learn event")
			// Not implemented
		case <-mSupport.ClickedCh:
			logging.Debug("Support event")
			// Not implemented
		case <-mAccount.ClickedCh:
			logging.Debug("Account event")
			// Not implemented
		case <-mProjects.ClickedCh:
			logging.Debug("Projects event")
			cancel()
			cancel, err = refreshProjects(model, mProjects)
			if err != nil {
				logging.Error("Could not refresh projects, got err: %v", err)
			}
		case <-mQuit.ClickedCh:
			logging.Debug("Quit event")
			systray.Quit()
			return nil
		}
	}
}

func refreshProjects(model *model.SvcModel, menuItem *systray.MenuItem) (context.CancelFunc, error) {
	logging.Debug("Refresh projects")
	ctx, cancel := context.WithCancel(context.Background())

	localProjects, err := model.LocalProjects()
	if err != nil {
		return cancel, errs.Wrap(err, "Could not get local project listing")
	}

	for _, project := range localProjects {
		mProject := menuItem.AddSubMenuItem(fmt.Sprintf("%s/%s", project.Owner, project.Name), "")
		go func(ctx context.Context, proj *graph.Project) {
			for {
				select {
				case <-mProject.ClickedCh:
					err = open.Prompt(fmt.Sprintf("state activate %s/%s --path %s", proj.Owner, proj.Name, proj.Locations[0]))
					if err != nil {
						logging.Error("Could not open local projects prompt for project %s/%s, got error: %v", proj.Owner, proj.Name, err)
					}
				case <-ctx.Done():
					return
				}
			}
		}(ctx, project)
	}

	return cancel, nil
}

func onExit() {
	// Not implemented
}
