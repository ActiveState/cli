package constants

// CfgSessionToken is the configuration key for the session token the installer sets
const CfgSessionToken = "sessionToken"

// CatRunCmd is the event category used for running commands
const CatRunCmd = "run-command"

// CatShim is the event category used for shimmed commands
const CatShim = "shim"

// CatBuild is the event category used for headchef builds
const CatBuild = "build"

// CatRuntimeDebug is the event category used for debugging runtime setup and usage.
// It should only be used to help diagnose where errors and dropoffs may be happening.
const CatRuntimeDebug = "runtime-debug"

// CatRuntimeUsage is the event category used for all runtime usage
const CatRuntimeUsage = "runtime-use"

// CatConfig is the event category used for all configuration events
const CatConfig = "config"

// CatUpdate is the event category used for all update events
const CatUpdates = "updates"

// CatInstaller is the event category used for installer events.
const CatInstaller = "installer"

// CatInstallerFunnel is the event category used for installer funnel events.
const CatInstallerFunnel = "installer-funnel"

// SrcStateTool is the event source for events sent by state.
const SrcStateTool = "State Tool"

// SrcStateService is the event source for events sent by state-svc.
const SrcStateService = "State Service"

// SrcStateInstaller is the event source for events sent by state-installer.
const SrcStateInstaller = "State Installer"

// SrcStateRemoteInstaller is the event source for events sent by state-remote-installer.
const SrcStateRemoteInstaller = "State Remote Installer"

// SrcOfflineInstaller is the event source for events sent by offline installers.
const SrcOfflineInstaller = "Offline Installer"

// SrcExecutor is the event source for events sent by executors.
const SrcExecutor = "Executor"

// ActRuntimeHeartbeat is the event action sent when a runtime is in use
const ActRuntimeHeartbeat = "heartbeat"

// ActRuntimeSuccess is the event action sent attempting to use a runtime
const ActRuntimeAttempt = "attempt"

// ActRuntimeStart is the event action sent when creating a runtime
const ActRuntimeStart = "start"

// ActRuntimeDelete is the event action sent when uninstalling a runtime
const ActRuntimeDelete = "delete"

// ActRuntimeCache is the event action sent when a runtime is constructed from the local cache alone
const ActRuntimeCache = "cache"

// ActRuntimeBuild is the event action sent when starting a remote build for the project
const ActRuntimeBuild = "build"

// ActRuntimeDownload is the event action sent before starting the download of artifacts for a runtime
const ActRuntimeDownload = "download"

// ActRuntimeSuccess is the event action sent when a runtime's environment has been successfully computed (for the first time)
const ActRuntimeSuccess = "success"

// ActRuntimeFailure is the event action sent when a failure occurred anytime during a runtime operation
const ActRuntimeFailure = "failure"

// ActConfigSet is the event action sent when a configuration value is set
const ActConfigSet = "set"

// ActConfigUnset is the event action sent when a configuration value is unset
const ActConfigUnset = "unset"

// ActConfigGet is the event action sent when determining if an update should be checked
const ActShouldUpdate = "should-autoupdate"

// ActConfigGet is the event action sent when an update is checked
const ActUpdateCheck = "update-check"

// ActUpdateDownload is the event action sent when an update retrieved
const ActUpdateDownload = "download"

// ActUpdateInstall is the event action sent when an update is installed
const ActUpdateInstall = "install"

// ActUpdateRelaunch is the event action sent after and update is installed and the state tool is relaunched
const ActUpdateRelaunch = "relaunch"

// LblRtFailUpdate is the label sent with an ActRuntimeFailure event if an error occurred during a runtime update
const LblRtFailUpdate = "update"

// CatPpmConversion is the event category used for ppm-conversion events
const CatPpmConversion = "ppm-conversion"

// CatPPMShimCmd is the event category used for PPM shim events
const CatPPMShimCmd = "ppm-shim"

// CatCommandExit is the event category used to track the success of state commands
const CatCommandExit = "command-exit"

// CatCommandExit is the event category used to track the error that was returned from a command
const CatCommandError = "command-error"

// CatActivationFlow is for events that outline the activation flow
const CatActivationFlow = "activation"

// CatPrompt is for prompt events
const CatPrompt = "prompt"

// CatMist is for miscellaneous events
const CatMisc = "misc"

// CatStateSvc is for state-svc events
const CatStateSvc = "state-svc"

// CatPackageOp is for `state packages` events
const CatPackageOp = "package-operation"

// CatOfflineInstaller is the event category used for all data relating to offline installs and uninstalls
const CatOfflineInstaller = "offline-installer"

// ActOfflineInstallerStart is the event action for the offline installer/uninstaller being initiated
const ActOfflineInstallerStart = "start"

// ActOfflineInstallerFailure is the event action for the offline installer/uninstaller failing
const ActOfflineInstallerFailure = "failure"

// ActOfflineInstallerStart is the event action for the offline installer/uninstaller succeeding
const ActOfflineInstallerSuccess = "success"

// ActOfflineInstallerAbort is the event action for the offline installer being terminated by the user
const ActOfflineInstallerAbort = "aborted"

// CatDebug is the event category used for all debug events
const CatDebug = "debug"

// ActCommandErorr is the event action used for command errors
const ActCommandError = "command-error"

// ActCommandInputError is the event action used for command input errors
const ActCommandInputError = "command-input-error"

// ActExecutorExit is the event action used for executor exit codes
const ActExecutorExit = "executor-exit"

// UpdateLabelSuccess is the sent if an auto-update was successful
const UpdateLabelSuccess = "success"

// UpdateLabelFailed is the sent if an auto-update failed
const UpdateLabelFailed = "failure"

// UpdateLabelTrue is the sent if we should auto-update
const UpdateLabelTrue = "true"

// UpdateLabelForward is the sent if we should not auto-update as we are forwarding a command
const UpdateLabelForward = "forward"

// UpdateLabelUnitTest is the sent if we should not auto-update as we are running unit tests
const UpdateLabelUnitTest = "unittest"

// UpdateLabelConflict is the sent if we should not auto-update as the current command might conflict
const UpdateLabelConflict = "conflict"

// UpdateLabelDisabledEnv is the sent if we should not auto-update as the user has disabled auto-updates via the environment
const UpdateLabelDisabledEnv = "disabled-env"

// UpdateLabelDisabledConfig is the sent if we should not auto-update as the user has disabled auto-updates via the config
const UpdateLabelDisabledConfig = "disabled-config"

// AutoUpdateLabelDisabledCI is the sent if we should not auto-update as we are on CI
const UpdateLabelCI = "ci"

// UpdateLabelFreshInstall is the sent if we should not auto-update as we are on a fresh install
const UpdateLabelFreshInstall = "fresh-install"

// UpdateLabelLocked is the sent if we should not auto-update as the state tool is locked
const UpdateLabelLocked = "locked"

// UpdateLabelTooFreq is the sent if we should not auto-update as the last check was too recent
const UpdateLabelTooFreq = "too-frequent"

// UpdateLabelAvailable is the sent if the update information is available
const UpdateLabelAvailable = "available"

// UpdateLabelUnavailable is the sent if the update information is unavailable
const UpdateLabelUnavailable = "unavailable"

// UpdateErrorInProgress is sent if an update is already in progress
const UpdateErrorInProgress = "Update already in progress"

// UpdateErrorInstallFailed is sent if an update failed at the install step
const UpdateErrorInstallFailed = "Could not install update"

// UpdateErrorExecutable is sent if the state executable could not be located
const UpdateErrorExecutable = "Could not locate state executable for relaunch"

// UpdateErrorRelaunch is sent if the updated state executable could not be relaunched
const UpdateErrorRelaunch = "Could not execute relaunch"

// UpdateErrorNotFound is sent if the update information could not be found
const UpdateErrorNotFound = "Update info could not be found"

// UpdateErrorBlocked is sent if the update information was blocked or the service was unavailable
const UpdateErrorBlocked = "Update info request blocked or service unavailable"

// UpdateErrorFetch is sent if the update information could not be fetched
const UpdateErrorFetch = "Could not fetch update info"

// UpdateErrorTempDir is sent if the temp dir for update unpacking could not be created
const UpdateErrorTempDir = "Could not create temp dir"

// UpdateErrorNoInstaller is sent if the downloaded update does not have an installer
const UpdateErrorNoInstaller = "Downloaded update does not have installer"

// UpdateErrorInstallPath is sent if the install path could not be detected
const UpdateErrorInstallPath = "Could not detect install path"

// CatInteractions is the event category used for tracking user interactions.
const CatInteractions = "interactions"

// ActVcsConflict is the event action sent when `state pull` results in a conflict.
const ActVcsConflict = "vcs-conflict"

// LabelVcsConflictMergeStrategyFailed is the label to use when a merge fails.
const LabelVcsConflictMergeStrategyFailed = "Failed"
