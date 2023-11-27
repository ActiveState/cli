package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"

	"github.com/ActiveState/cli/internal/analytics"
	ac "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/offinstall"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

const artifactsTarGZName = "artifacts.tar.gz"
const assetsPathName = "assets"
const artifactsPathName = "artifacts"
const licenseFileName = "LICENSE.txt"
const installerConfigFileName = "installer_config.json"
const uninstallerFileNameRoot = "uninstall" + exeutils.Extension

type runner struct {
	out       output.Outputer
	prompt    prompt.Prompter
	analytics analytics.Dispatcher
	cfg       *config.Instance
	shell     subshell.SubShell
	icfg      InstallerConfig
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Analyticer
	primer.Configurer
	primer.Subsheller
}

func NewRunner(prime primeable) *runner {
	return &runner{
		prime.Output(),
		prime.Prompt(),
		prime.Analytics(),
		prime.Config(),
		prime.Subshell(),
		InstallerConfig{},
	}
}

type InstallerConfig struct {
	OrgName     string `json:"org_name"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	CommitID    string `json:"commit_id"`
}

type Params struct {
	path string
}

func newParams() *Params {
	return &Params{}
}

func (r *runner) Run(params *Params) (rerr error) {
	var installerDimensions *dimensions.Values
	defer func() {
		if rerr == nil {
			return
		}
		if locale.IsInputError(rerr) {
			r.analytics.EventWithLabel(ac.CatOfflineInstaller, ac.ActOfflineInstallerAbort, errs.JoinMessage(rerr), installerDimensions)
		} else {
			r.analytics.EventWithLabel(ac.CatOfflineInstaller, ac.ActOfflineInstallerFailure, errs.JoinMessage(rerr), installerDimensions)
		}
	}()

	tempDir, err := ioutil.TempDir("", "artifacts-")
	if err != nil {
		return errs.Wrap(err, "Unable to create temporary directory")
	}
	defer os.RemoveAll(tempDir)

	/* Extract Assets */
	backpackZipFile := os.Args[0]
	assetsPath := filepath.Join(tempDir, assetsPathName)
	if err := r.extractAssets(assetsPath, backpackZipFile); err != nil {
		return errs.Wrap(err, "Could not extract assets")
	}

	err = r.prepareInstallerConfig(assetsPath)
	if err != nil {
		return errs.Wrap(err, "Could not read installer config, this installer appears to be corrupted.")
	}

	namespace := project.NewNamespace(r.icfg.OrgName, r.icfg.ProjectName, "")
	installerDimensions = &dimensions.Values{
		ProjectNameSpace: ptr.To(namespace.String()),
		CommitID:         &r.icfg.CommitID,
		Trigger:          ptr.To(target.TriggerOfflineInstaller.String()),
	}
	r.analytics.Event(ac.CatOfflineInstaller, "start", installerDimensions)

	// Detect target path
	targetPath, err := r.getTargetPath(params.path)
	if err != nil {
		return errs.Wrap(err, "Could not determine target path")
	}

	/* Validate Target Path */
	if err := r.validateTargetPath(targetPath); err != nil {
		return errs.Wrap(err, "Could not validate target path")
	}

	/* Prompt for License */
	accepted, err := r.promptLicense(assetsPath)
	if err != nil {
		return errs.Wrap(err, "Could not prompt for license")
	}
	if !accepted {
		return locale.NewInputError("License not accepted")
	}

	/* Extract Artifacts */
	artifactsPath := filepath.Join(tempDir, artifactsPathName)
	if err := r.extractArtifacts(artifactsPath, assetsPath); err != nil {
		return errs.Wrap(err, "Could not extract artifacts")
	}

	/* Install Artifacts */
	asrt, err := r.setupRuntime(artifactsPath, targetPath)
	if err != nil {
		return errs.Wrap(err, "Could not setup runtime")
	}

	/* Manually Install License File */
	{
		err = fileutils.CopyFile(filepath.Join(assetsPath, licenseFileName), filepath.Join(targetPath, licenseFileName))
		if err != nil {
			return errs.Wrap(err, "Error copying license file")
		}
	}

	/* Manually Install config File */
	{
		err = fileutils.CopyFile(
			filepath.Join(assetsPath, installerConfigFileName),
			filepath.Join(targetPath, installerConfigFileName),
		)
		if err != nil {
			return errs.Wrap(err, "Error copying config file")
		}
	}

	var uninstallerSrc string
	var uninstallerDest string

	/* Manually Install uninstaller */
	if rt.GOOS == "windows" {
		/* shenanigans because windows won't let you delete an executable that's running */
		installDir, err := filepath.Abs(targetPath)
		if err != nil {
			return errs.Wrap(err, "Error determining absolute install directory")
		}
		uninstallDir := filepath.Join(installDir, "uninstall-data")
		if fileutils.DirExists(uninstallDir) {
			if err := os.RemoveAll(uninstallDir); err != nil {
				return errs.Wrap(err, "Error removing uninstall directory")
			}
		}
		if err := os.Mkdir(uninstallDir, os.ModeDir); err != nil {
			return errs.Wrap(err, "Error creating uninstall directory")
		}

		uninstallerSrc = filepath.Join(assetsPath, uninstallerFileNameRoot)
		uninstallerDest = filepath.Join(uninstallDir, uninstallerFileNameRoot)

		// create batch script which copies the uninstaller to a temp dir and runs it from there this is necessary
		// because windows won't let you delete an executable that's running
		// The last message about ignoring the error is because the uninstaller will delete the directory the batch file
		// is in, which unlike with the exe is fine because batch files are "special", but it does result in a benign
		// "File not Found" error
		batch := fmt.Sprintf(
			`
				@echo off
				copy %[1]s\%[2]s %%TEMP%%\%[2]s >nul 2>&1
				%%TEMP%%\%[2]s %[3]s
				del %%TEMP%%\%[2]s >nul 2>&1
				echo You can safely ignore any File not Found errors following this message.
				`,
			uninstallDir,
			uninstallerFileNameRoot,
			installDir,
		)
		err = os.WriteFile(filepath.Join(installDir, "uninstall.bat"), []byte(batch), 0755)
		if err != nil {
			return errs.Wrap(err, "Error creating uninstall script")
		}
	} else {
		uninstallerSrc = filepath.Join(assetsPath, uninstallerFileNameRoot)
		uninstallerDest = filepath.Join(targetPath, uninstallerFileNameRoot)
	}
	{
		if fileutils.TargetExists(uninstallerDest) {
			err := os.Remove(uninstallerDest)
			if err != nil {
				return errs.Wrap(err, "Error removing existing uninstaller")
			}
		}
		err = fileutils.CopyFile(
			uninstallerSrc,
			uninstallerDest,
		)
		if err != nil {
			return errs.Wrap(err, "Error copying uninstaller")
		}
		err = os.Chmod(uninstallerDest, 0555)
		if err != nil {
			return errs.Wrap(err, "Error making uninstaller executable")
		}
	}

	/* Configure Environment */
	if err := r.configureEnvironment(targetPath, namespace, asrt); err != nil {
		return errs.Wrap(err, "Could not configure environment")
	}

	r.analytics.Event(ac.CatOfflineInstaller, ac.ActOfflineInstallerSuccess, installerDimensions)

	r.out.Print(fmt.Sprintf(`Installation complete.
Your language runtime has been installed in [ACTIONABLE]%s[/RESET].`, targetPath))

	return nil
}

func (r *runner) prepareInstallerConfig(assetsPath string) error {
	icfg := InstallerConfig{}
	installerConfigPath := filepath.Join(assetsPath, installerConfigFileName)
	configData, err := os.ReadFile(installerConfigPath)
	if err != nil {
		return errs.Wrap(err, "Failed to read config_file")
	}
	if err := json.Unmarshal(configData, &icfg); err != nil {
		return errs.Wrap(err, "Failed to decode config_file")
	}

	if icfg.ProjectName == "" {
		return errs.New("ProjectName is empty")
	}

	if icfg.OrgName == "" {
		return errs.New("OrgName is empty")
	}

	if icfg.CommitID == "" {
		return errs.New("CommitID is empty")
	}

	r.icfg = icfg

	return nil
}

func (r *runner) setupRuntime(artifactsPath string, targetPath string) (*runtime.Runtime, error) {
	logfile, err := buildlogfile.New(outputhelper.NewCatcher())
	if err != nil {
		return nil, errs.Wrap(err, "Unable to create new logfile object")
	}

	ns := project.NewNamespace(r.icfg.OrgName, r.icfg.ProjectName, r.icfg.CommitID)
	offlineTarget := target.NewOfflineTarget(ns, targetPath, artifactsPath)
	offlineTarget.SetTrigger(target.TriggerOfflineInstaller)

	offlineProgress := newOfflineProgressOutput(r.out)
	eventHandler := events.NewRuntimeEventHandler(offlineProgress, nil, logfile)

	rti, err := runtime.New(offlineTarget, r.analytics, nil, nil)
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return nil, errs.Wrap(err, "Could not create runtime")
		}
		if err = rti.Update(eventHandler); err != nil {
			return nil, errs.Wrap(err, "Had an installation error")
		}
	}
	return rti, nil
}

func (r *runner) extractArtifacts(artifactsPath, assetsPath string) error {
	if err := os.Mkdir(artifactsPath, os.ModePerm); err != nil {
		return errs.Wrap(err, "Unable to create artifactsPath directory")
	}

	archivePath := filepath.Join(assetsPath, artifactsTarGZName)
	ua := unarchiver.NewTarGz()
	f, siz, err := ua.PrepareUnpacking(archivePath, artifactsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to prepare unpacking of artifact tarball")
	}

	pb := mpb.New(
		mpb.WithWidth(40),
	)
	barName := "Extracting"
	bar := pb.AddBar(
		siz,
		mpb.PrependDecorators(decor.Name(barName, decor.WC{W: len(barName) + 1, C: decor.DidentRight})),
	)

	ua.SetNotifier(func(filename string, _ int64, isDir bool) {
		if !isDir {
			bar.Increment()
		}
	})

	err = ua.Unarchive(f, siz, artifactsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to unarchive artifacts to artifactsPath")
	}

	bar.SetTotal(0, true)
	bar.Abort(true)
	pb.Wait()

	return nil
}

func (r *runner) extractAssets(assetsPath string, backpackZipFile string) error {
	if err := os.Mkdir(assetsPath, os.ModePerm); err != nil {
		return errs.Wrap(err, "Unable to create assetsPath")
	}

	ua := unarchiver.NewZip()
	f, siz, err := ua.PrepareUnpacking(backpackZipFile, assetsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to prepare unpacking of backpack")
	}

	err = ua.Unarchive(f, siz, assetsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to unarchive Assets to assetsPath")
	}

	return nil
}

func (r *runner) configureEnvironment(path string, namespace *project.Namespaced, asrt *runtime.Runtime) error {
	env, err := asrt.Env(false, false)
	if err != nil {
		return errs.Wrap(err, "Error setting environment")
	}

	if rt.GOOS == "windows" {
		contents, err := assets.ReadFileBytes("scripts/setenv.bat")
		if err != nil {
			return errs.Wrap(err, "Error reading file bytes")
		}
		err = fileutils.WriteFile(filepath.Join(path, "setenv.bat"), contents)
		if err != nil {
			return locale.WrapError(err,
				"err_deploy_write_setenv",
				"Could not create setenv batch scriptfile at path: {{.V0}}",
				path)
		}
	}

	// Configure available shells
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}

	id := sscommon.ProjectRCIdentifier(sscommon.OfflineInstallID, namespace)
	err = subshell.ConfigureAvailableShells(r.shell, r.cfg, env, id, !isAdmin)
	if err != nil {
		return locale.WrapError(err,
			"err_deploy_subshell_write",
			"Could not write environment information to your shell configuration.")
	}

	binPath := filepath.Join(path, "bin")
	if err := fileutils.MkdirUnlessExists(binPath); err != nil {
		return locale.WrapError(err, "err_deploy_binpath", "Could not create bin directory.")
	}

	// Write global env file
	err = r.shell.SetupShellRcFile(binPath, env, nil)
	if err != nil {
		return locale.WrapError(err, "err_deploy_subshell_rc_file", "Could not create environment script.")
	}

	return nil
}

func (r *runner) getTargetPath(inputPath string) (string, error) {
	var targetPath string
	if inputPath != "" {
		targetPath = inputPath
	} else {
		parentDir, err := offinstall.DefaultInstallParentDir()
		if err != nil {
			return "", errs.Wrap(err, "Could not determine default install path")
		}
		targetPath = filepath.Join(parentDir, r.icfg.ProjectName)

		targetPath, err = r.prompt.Input("", "Enter an installation directory", &targetPath)
		if err != nil {
			return "", errs.Wrap(err, "Could not retrieve installation directory")
		}
	}
	return targetPath, nil
}

func (r *runner) validateTargetPath(path string) error {
	if !fileutils.IsWritable(path) {
		return errs.New(
			"Cannot write to [ACTIONABLE]%s[/RESET]. Please ensure that the directory is writeable without "+
				"needing admin privileges or run this installer with Admin.", path)
	}

	if fileutils.TargetExists(path) {
		if !fileutils.IsDir(path) {
			return errs.New("Target path [ACTIONABLE]%s[/RESET] is not a directory", path)
		}

		empty, err := fileutils.IsEmptyDir(path)
		if err != nil {
			return errs.Wrap(err, "Test for directory empty failed")
		}
		if !empty {
			installNonEmpty, err := r.prompt.Confirm(
				"Setup",
				"Installation directory is not empty, install anyway?",
				ptr.To(true))
			if err != nil {
				return errs.Wrap(err, "Unable to get confirmation to install into non-empty directory")
			}

			if !installNonEmpty {
				return locale.NewInputError(
					"offline_installer_err_installdir_notempty",
					"Installation directory ({{.V0}}) not empty, installation aborted",
					path)
			}
		}
	}

	return nil
}

func (r *runner) promptLicense(assetsPath string) (bool, error) {
	licenseFileAssetPath := filepath.Join(assetsPath, licenseFileName)
	licenseContents, err := fileutils.ReadFile(licenseFileAssetPath)
	if err != nil {
		return false, errs.Wrap(err, "Unable to open License file")
	}
	r.out.Print(licenseContents)

	choice, err := r.prompt.Confirm("", "Do you accept the ActiveState Runtime Installer License Agreement?", ptr.To(false))
	if err != nil {
		return false, err
	}

	if err != nil {
		return false, errs.Wrap(err, "Unable to confirm license")
	}

	return choice, nil
}
