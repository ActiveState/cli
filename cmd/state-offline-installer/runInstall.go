package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"

	"github.com/ActiveState/cli/internal/analytics/client/blackhole"
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/buildlogfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/cmdlets/prompts"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

const artifactsTarGZName = "artifacts.tar.gz"
const assetsPathName = "assets"
const artifactsPathName = "artifacts"
const licenseFileName = "LICENSE.txt"

func runInstall(out output.Outputer, params *Params) error {

	analytics := blackhole.New()
	prompt := prompt.New(true, analytics)
	default_boolean_answer := true

	tempDir, err := ioutil.TempDir("", "artifacts-")
	if err != nil {
		return errs.Wrap(err, "Unable to create temporary directory")
	}

	defer os.RemoveAll(tempDir)

	out.Print(fmt.Sprintf("Temp directory is: %s", tempDir))

	installToDir := params.path
	assetsPath := filepath.Join(tempDir, assetsPathName)
	artifactsPath := filepath.Join(tempDir, artifactsPathName)
	licenseFilePath := filepath.Join(assetsPath, licenseFileName)

	// Double check if installation directory exists already
	if fileutils.DirExists(installToDir) {
		empty, err := fileutils.IsEmptyDir(installToDir)
		if err != nil {
			return errs.Wrap(err, "Test for directory empty failed")
		}
		if !empty {
			installNonEmpty, err := prompt.Confirm("Setup", "Installation directory is not empty, install anyway?", &default_boolean_answer)
			if err != nil {
				return errs.Wrap(err, "Unable to get confirmation to install into non-empty directory")
			}

			if !installNonEmpty {
				return locale.NewInputError("offline_installer_err_installdir_notempty", "Installation directory ({{.V0}}) not empty, installation aborted", installToDir)
			}
		}
	}

	if err := os.Mkdir(assetsPath, os.ModePerm); err != nil {
		return errs.Wrap(err, "Unable to create assetsPath")
	}

	if err := os.Mkdir(artifactsPath, os.ModePerm); err != nil {
		return errs.Wrap(err, "Unable to create artifactsPath directory")
	}

	ua := unarchiver.NewZip()
	out.Print(fmt.Sprintf("Stage 1 of 3 Start: Decompressing assets into: %s", assetsPath))
	f, siz, err := ua.PrepareUnpacking(params.backpackZipFile, assetsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to prepare unpacking of backpack")
	}

	err = ua.Unarchive(f, siz, assetsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to unarchive Assets to assetsPath")
	}

	out.Print(fmt.Sprintf("Stage 1 of 3 Finished: Decompressing assets into: %s", assetsPath))

	tos := prompts.NewOfflineFileTOS(licenseFilePath)
	accepted, err := prompts.PromptTOS(tos, out, prompt)
	if err != nil {
		return errs.Wrap(err, "Error with TOS acceptance")
	}
	if !accepted {
		return locale.NewInputError("tos_not_accepted", "")
	}

	archivePath := filepath.Join(assetsPath, artifactsTarGZName)

	out.Print(fmt.Sprintf("Stage 2 of 3 Start: Decompressing artifacts into: %s", artifactsPath))
	ua = unarchiver.NewTarGz()
	f, siz, err = ua.PrepareUnpacking(archivePath, artifactsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to prepare unpacking of artifact tarball")
	}

	ua.SetNotifier(func(filename string, _ int64, isDir bool) {
		if !isDir {
			out.Print(fmt.Sprintf("Unpacking artifact %s", filename))
		}
	})

	err = ua.Unarchive(f, siz, artifactsPath)
	if err != nil {
		return errs.Wrap(err, "Unable to unarchive artifacts to artifactsPath")
	}

	out.Print(fmt.Sprintf("Stage 2 of 3 Finished: Decompressing artifacts into: %s", artifactsPath))

	out.Print(fmt.Sprintf("Stage 3 of 3 Start: Installing artifacts from: %s", artifactsPath))
	offlineTarget := target.NewOfflineTarget(installToDir, artifactsPath)

	offlineProgress := newOfflineProgressOutput(out)
	logfile, err := buildlogfile.New(outputhelper.NewCatcher())
	if err != nil {
		return errs.Wrap(err, "Unable to create new logfile object")
	}
	eventHandler := events.NewRuntimeEventHandler(offlineProgress, nil, logfile)

	rti, err := runtime.New(offlineTarget, analytics, nil)
	if err != nil && runtime.IsNeedsUpdateError(err) {
		err = rti.Update(nil, eventHandler)
		if err != nil {
			return errs.Wrap(err, "Had an installation error")
		}
	}
	out.Print(fmt.Sprintf("Stage 3 of 3 Finished: Installing artifacts from: %s", artifactsPath))

	configureEnvironmentAccepted, err := prompt.Confirm("Setup", "Setup environment for installed project?", &default_boolean_answer)
	if err != nil {
		return errs.Wrap(err, "Error getting confirmation")
	}

	if configureEnvironmentAccepted {
		// This is the code to setup the environment when we install...
		cfg, err := config.New()
		if err != nil {
			return errs.Wrap(err, "Error configuring environment")
		}

		env, err := rti.Env(false, false)
		if err != nil {
			return errs.Wrap(err, "Error setting environment")
		}

		sshell := subshell.New(cfg)

		if rt.GOOS == "windows" {
			contents, err := assets.ReadFileBytes("scripts/setenv.bat")
			if err != nil {
				return errs.Wrap(err, "Error reading file bytes")
			}
			err = fileutils.WriteFile(filepath.Join(installToDir, "setenv.bat"), contents)
			if err != nil {
				return locale.WrapError(err, "err_deploy_write_setenv", "Could not create setenv batch scriptfile at path: {{.V0}}", installToDir)
			}
		}

		err = sshell.WriteUserEnv(cfg, env, sscommon.DeployID, true)
		if err != nil {
			return locale.WrapError(err, "err_deploy_subshell_write", "Could not write environment information to your shell configuration.")
		}

		binPath := filepath.Join(offlineTarget.Dir(), "bin")
		if err := fileutils.MkdirUnlessExists(binPath); err != nil {
			return locale.WrapError(err, "err_deploy_binpath", "Could not create bin directory.")
		}

		namespace_to_use := project.NewNamespace("owner", "project", "commitID")

		// Write global env file
		err = sshell.SetupShellRcFile(binPath, env, *namespace_to_use)
		if err != nil {
			return locale.WrapError(err, "err_deploy_subshell_rc_file", "Could not create environment script.")
		}
	}

	// Copy license file
	err = fileutils.CopyFile(licenseFilePath, filepath.Join(installToDir, "LICENSE.txt"))
	if err != nil {
		return errs.Wrap(err, "Error copying file")
	}

	out.Print("Runtime installation completed.")

	return nil
}
