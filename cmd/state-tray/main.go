package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/getlantern/systray"
	"github.com/gobuffalo/packr"
	"github.com/rollbar/rollbar-go"
	"github.com/shirou/gopsutil/process"

	"github.com/ActiveState/cli/cmd/state-tray/internal/menu"
	"github.com/ActiveState/cli/cmd/state-tray/internal/open"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/pkg/platform/model"
)

const (
	assetsPath = "../../assets"
	iconFile   = "icon.ico"
)

func main() {
	verbose := os.Getenv("VERBOSE") != ""

	logging.CurrentHandler().SetVerbose(verbose)
	logging.SetupRollbar(constants.StateTrayRollbarToken)

	systray.Run(onReady, onExit)
}

func onReady() {
	var exitCode int
	defer exit(exitCode)

	err := run()
	if err != nil {
		errMsg := errs.Join(err, ": ").Error()
		logging.Error("Systray encountered an error: %v", errMsg)
		fmt.Fprintln(os.Stderr, errMsg)
		exitCode = 1
	}
}

func run() error {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not get new config instance")
	}

	currentPID, err := trayPID(cfg)
	if err == nil && currentPID != nil {
		return errs.New("ActiveState Desktop is already running")
	}

	if err := cfg.Set(config.ConfigKeyTrayPid, os.Getpid()); err != nil {
		return errs.Wrap(err, "Could not write pid to config file.")
	}

	box := packr.NewBox(assetsPath)
	systray.SetIcon(box.Bytes(iconFile))

	svcm := svcmanager.New(cfg)
	if err := svcm.StartAndWait(); err != nil {
		return errs.Wrap(err, "Service failed to start")
	}

	model, err := model.NewSvcModel(context.Background(), cfg)
	if err != nil {
		return errs.Wrap(err, "Could not create new service model")
	}

	systray.SetTooltip(locale.Tl("tray_tooltip", constants.TrayAppName))

	mUpdate := systray.AddMenuItem(
		locale.Tl("tray_update_title", "Update Available"),
		locale.Tl("tray_update_tooltip", "Update your ActiveState Desktop installation"),
	)
	mUpdate.Hide()

	updNotice := updateNotice{
		box:  box,
		item: mUpdate,
	}
	closeUpdateSupervision := superviseUpdate(model, &updNotice)
	defer closeUpdateSupervision()

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
		locale.Tl("tray_blog_title", "Blog"),
		locale.Tl("tray_blog_tooltip", "Open ActiveState blog"),
	)
	mSupport := mPlatform.AddSubMenuItem(
		locale.Tl("tray_support_title", "Support"),
		locale.Tl("tray_support_tooltip", "Open support page"),
	)
	mAccount := mPlatform.AddSubMenuItem(
		locale.Tl("tray_account_title", "Account"),
		locale.Tl("tray_account_tooltip", "Open your account page"),
	)

	trayInfo := appinfo.TrayApp()

	systray.AddSeparator()
	as := autostart.New(trayInfo.Name(), trayInfo.Exec())
	enabled, err := as.IsEnabled()
	if err != nil {
		return errs.Wrap(err, "Could not check if app autostart is enabled")
	}
	mAutoStart := systray.AddMenuItemCheckbox(
		locale.Tl("tray_autostart", "Start on Login"), "", enabled,
	)
	systray.AddSeparator()

	mProjects := systray.AddMenuItem(locale.Tl("tray_projects_title", "Local Projects"), "")
	mReload := mProjects.AddSubMenuItem("Reload", "Reload the local projects listing")
	localProjectsUpdater := menu.NewLocalProjectsUpdater(mProjects)

	localProjects, err := model.LocalProjects()
	if err != nil {
		logging.Error("Could not get local projects listing: %v", err)
	}
	localProjectsUpdater.Update(localProjects)

	systray.AddSeparator()

	mQuit := systray.AddMenuItem(locale.Tl("tray_exit", "Exit"), "")

	for {
		select {
		case <-mAbout.ClickedCh:
			logging.Debug("About event")
			err = open.TerminalAndWait(appinfo.StateApp().Exec() + " --version")
			if err != nil {
				logging.Error("Could not open command prompt: %v", err)
			}
		case <-mDoc.ClickedCh:
			logging.Debug("Documentation event")
			err = open.Browser(constants.DocumentationURL)
			if err != nil {
				logging.Error("Could not open documentation url: %v", err)
			}
		case <-mLearn.ClickedCh:
			logging.Debug("Learn event")
			err = open.Browser(constants.ActiveStateBlogURL)
			if err != nil {
				logging.Error("Could not open blog url: %v", err)
			}
		case <-mSupport.ClickedCh:
			logging.Debug("Support event")
			err = open.Browser(constants.ActiveStateSupportURL)
			if err != nil {
				logging.Error("Could not open support url: %v", err)
			}
		case <-mAccount.ClickedCh:
			logging.Debug("Account event")
			err = open.Browser(constants.ActiveStateAccountURL)
			if err != nil {
				logging.Error("Could not open account url: %v", err)
			}
		case <-mReload.ClickedCh:
			logging.Debug("Projects event")
			localProjects, err = model.LocalProjects()
			if err != nil {
				logging.Error("Could not get local projects listing: %v", err)
			}
			localProjectsUpdater.Update(localProjects)
		case <-mAutoStart.ClickedCh:
			logging.Debug("Autostart event")
			var err error
			enabled, err := as.IsEnabled()
			if err != nil {
				logging.Error("Could not check if autostart is enabled: %v", err)
			}
			if enabled {
				logging.Debug("Disable")
				err = as.Disable()
				if err == nil {
					mAutoStart.Uncheck()
				}
			} else {
				logging.Debug("Enable")
				err = as.Enable()
				if err == nil {
					mAutoStart.Check()
				}
			}
			if err != nil {
				logging.Error("Could not toggle autostart tray: %v", errs.Join(err, ": "))
			}
		case <-mUpdate.ClickedCh:
			logging.Debug("Update event")
			updlgInfo := appinfo.UpdateDialogApp()
			if err := execute(updlgInfo.Exec(), nil); err != nil {
				return errs.New("Could not execute: %s", updlgInfo.Name())
			}
		case <-mQuit.ClickedCh:
			logging.Debug("Quit event")
			systray.Quit()
		}
	}
}

func onExit() {
	// Not implemented
}

func exit(code int) {
	events.WaitForEvents(1*time.Second, rollbar.Close)
	os.Exit(code)
}

func execute(exec string, args []string) error {
	if !fileutils.FileExists(exec) {
		return errs.New("Could not find: %s", exec)
	}

	if _, err := exeutils.ExecuteAndForget(exec, args); err != nil {
		return errs.Wrap(err, "Could not start %s", exec)
	}

	return nil
}

func trayPID(cfg *config.Instance) (*int, error) {
	pid := cfg.GetInt(config.ConfigKeyTrayPid)
	if pid <= 0 {
		return nil, nil
	}

	pidExists, err := process.PidExists(int32(pid))
	if err != nil {
		return nil, errs.Wrap(err, "Could not verify if pid exists")
	}
	if !pidExists {
		return nil, nil
	}

	return &pid, nil
}
