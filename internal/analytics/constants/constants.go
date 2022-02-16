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

// ActRuntimeHeartbeat is the event action sent when a runtime is in use
const ActRuntimeHeartbeat = "heartbeat"

// ActRuntimeStart is the event action sent when creating a runtime
const ActRuntimeStart = "start"

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
