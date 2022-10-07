package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/runbits/buildlogfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/cmdlets/legalprompt"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

const artifactsTarGZName = "artifacts.tar.gz"
const assetsPathName = "assets"
const artifactsPathName = "artifacts"
const licenseFileName = "LICENSE.txt"

type runner struct {
	out       output.Outputer
	prompt    prompt.Prompter
	analytics analytics.Dispatcher
	cfg       *config.Instance
	shell     subshell.SubShell
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
	}
}

type Params struct {
	path string
}

func newParams() *Params {
	return &Params{path: "/tmp"}
}

func (r *runner) Run(params *Params) error {
	tempDir, err := ioutil.TempDir("", "artifacts-")
	if err != nil {
		return errs.Wrap(err, "Unable to create temporary directory")
	}
	defer os.RemoveAll(tempDir)

	r.out.Print(fmt.Sprintf("Temp directory is: %s", tempDir))
	r.out.Print(fmt.Sprintf("Installation directory is: %s", params.path))

	/* Validate Target Path */
	if err := r.validateTargetPath(params.path); err != nil {
		return errs.Wrap(err, "Could not validate target path")
	}

	/* Extract Assets */
	backpackZipFile := os.Args[0]
	assetsPath := filepath.Join(tempDir, assetsPathName)
	if err := r.extractAssets(assetsPath, backpackZipFile); err != nil {
		return errs.Wrap(err, "Could not extract assets")
	}

	/* Prompt for License */
	licenseFileAssetPath := filepath.Join(assetsPath, licenseFileName)
	{
		b, err := fileutils.ReadFile(licenseFileAssetPath)
		if err != nil {
			return errs.Wrap(err, "Unable to open License file")
		}

		accepted, err := legalprompt.CustomLicense(string(b), r.out, r.prompt)
		if err != nil {
			return errs.Wrap(err, "Error with license acceptance")
		}
		if !accepted {
			return locale.NewInputError("License not accepted")
		}
	}

	/* Extract Artifacts */
	artifactsPath := filepath.Join(tempDir, artifactsPathName)
	if err := r.extractArtifacts(artifactsPath, assetsPath); err != nil {
		return errs.Wrap(err, "Could not extract artifacts")
	}

	/* Install Artifacts */
	offlineTarget := target.NewOfflineTarget(params.path, artifactsPath)
	asrt, err := r.setupRuntime(artifactsPath, offlineTarget)
	if err != nil {
		return errs.Wrap(err, "Could not setup runtime")
	}

	/* Manually Install License File */
	{
		err = fileutils.CopyFile(licenseFileAssetPath, filepath.Join(params.path, licenseFileName))
		if err != nil {
			return errs.Wrap(err, "Error copying license file")
		}
	}

	/* Configure Environment */
	if err := r.configureEnvironment(params.path, asrt); err != nil {
		return errs.Wrap(err, "Could not configure environment")
	}

	r.out.Print("Runtime installation completed.")

	return nil
}

func (r *runner) setupRuntime(artifactsPath string, offlineTarget *target.OfflineTarget) (*runtime.Runtime, error) {
	r.out.Print(fmt.Sprintf("Stage 3 of 3 Start: Installing artifacts from: %s", artifactsPath))

	offlineProgress := newOfflineProgressOutput(r.out)
	logfile, err := buildlogfile.New(outputhelper.NewCatcher())
	if err != nil {
		return nil, errs.Wrap(err, "Unable to create new logfile object")
	}
	eventHandler := events.NewRuntimeEventHandler(offlineProgress, nil, logfile)

	rti, err := runtime.New(offlineTarget, r.analytics, nil, nil)
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return nil, errs.Wrap(err, "Could not create runtime")
		}
		if err = rti.Update(nil, eventHandler); err != nil {
			return nil, errs.Wrap(err, "Had an installation error")
		}
	}
	r.out.Print(fmt.Sprintf("Stage 3 of 3 Finished: Installing artifacts from: %s", artifactsPath))
	return rti, nil
}

func (r *runner) extractArtifacts(artifactsPath, assetsPath string) error {
	if err := os.Mkdir(artifactsPath, os.ModePerm); err != nil {
		return errs.Wrap(err, "Unable to create artifactsPath directory")
	}

	r.out.Print(fmt.Sprintf("Stage 2 of 3 Start: Decompressing artifacts into: %s", artifactsPath))
	archivePath := filepath.Join(assetsPath, artifactsTarGZName)
	ua := unarchiver.NewTarGz()
	f, siz, err := ua.PrepareUnpacking(archivePath, artifactsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to prepare unpacking of artifact tarball")
	}

	ua.SetNotifier(func(filename string, _ int64, isDir bool) {
		if !isDir {
			r.out.Print(fmt.Sprintf("Unpacking artifact %s", filename))
		}
	})

	err = ua.Unarchive(f, siz, artifactsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to unarchive artifacts to artifactsPath")
	}

	r.out.Print(fmt.Sprintf("Stage 2 of 3 Finished: Decompressing artifacts into: %s", artifactsPath))

	return nil
}

func (r *runner) extractAssets(assetsPath string, backpackZipFile string) error {
	if err := os.Mkdir(assetsPath, os.ModePerm); err != nil {
		return errs.Wrap(err, "Unable to create assetsPath")
	}

	ua := unarchiver.NewZip()
	r.out.Print(fmt.Sprintf("Stage 1 of 3 Start: Decompressing assets into: %s", assetsPath))
	f, siz, err := ua.PrepareUnpacking(backpackZipFile, assetsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to prepare unpacking of backpack")
	}

	err = ua.Unarchive(f, siz, assetsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to unarchive Assets to assetsPath")
	}

	r.out.Print(fmt.Sprintf("Stage 1 of 3 Finished: Decompressing assets into: %s", assetsPath))
	return nil
}

func (r *runner) configureEnvironment(path string, asrt *runtime.Runtime) error {
	configureEnvironmentAccepted, err := r.prompt.Confirm("Setup", "Setup environment for installed project?", p.BoolP(true))
	if err != nil {
		return errs.Wrap(err, "Error getting confirmation")
	}

	if !configureEnvironmentAccepted {
		return nil
	}
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

	err = r.shell.WriteUserEnv(r.cfg, env, sscommon.OfflineInstallID, true)
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

func (r *runner) validateTargetPath(path string) error {
	if !fileutils.DirExists(path) {
		return nil
	}

	empty, err := fileutils.IsEmptyDir(path)
	if err != nil {
		return errs.Wrap(err, "Test for directory empty failed")
	}
	if empty {
		return nil
	}

	installNonEmpty, err := r.prompt.Confirm(
		"Setup",
		"Installation directory is not empty, install anyway?",
		p.BoolP(true))
	if err != nil {
		return errs.Wrap(err, "Unable to get confirmation to install into non-empty directory")
	}

	if !installNonEmpty {
		return locale.NewInputError(
			"offline_installer_err_installdir_notempty",
			"Installation directory ({{.V0}}) not empty, installation aborted",
			path)
	}

	return nil
}
