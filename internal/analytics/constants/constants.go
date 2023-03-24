package constants

// CfgSessionToken is the configuration key for the session token the installer sets
const CfgSessionToken = "sessionToken"

// CatRunCmd is the event category used for running commands
const CatRunCmd = "run-command"

// CatShim is the event category used for shimmed commands
const CatShim = "shim"

// CatBuild is the event category used for headchef builds
const CatBuild = "build"

// CatRuntime is the event category used for all runtime setup and usage
const CatRuntime = "runtime"

// CatRuntimeUsage is the event category used for all runtime usage
const CatRuntimeUsage = "runtime-use"

// CatConfig is the event category used for all configuration events
const CatConfig = "config"

// CatUpdate is the event category used for all update events
const CatUpdates = "updates"

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

// ActRuntimeUserFailure is the event action sent when a user failure occurred anytime during a runtime operation
const ActRuntimeUserFailure = "user_failure"

// ActConfigSet is the event action sent when a configuration value is set
const ActConfigSet = "set"

// ActConfigUnset is the event action sent when a configuration value is unset
const ActConfigUnset = "unset"

// ActConfigGet is the event action sent when determining if an update should be checked
const ActShouldUpdate = "should-update"

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

// LblRtFailEnv is the label sent with  an ActRuntimeFailure event if an error occurred during the resolution of the runtime environment
const LblRtFailEnv = "env"

// CatPpmConversion is the event category used for ppm-conversion events
const CatPpmConversion = "ppm-conversion"

// ActBuildProject is the event action for requesting a build for a specific project
const ActBuildProject = "project"

// CatPPMShimCmd is the event category used for PPM shim events
const CatPPMShimCmd = "ppm-shim"

// CatTutorial is the event category used for tutorial level events
const CatTutorial = "tutorial"

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

// ActInputError is the event action used for input errors
const ActInputError = "input-error"
