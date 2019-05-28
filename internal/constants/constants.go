package constants

// VersionNumber holds the current version of our cli
const VersionNumber = "0.2.2"

// LibraryName contains the main name of this library
const LibraryName = "cli"

// LibraryNamespace is the namespace that the library belongs to
const LibraryNamespace = "github.com/ActiveState/"

// CommandName holds the name of our command
const CommandName = "state"

// ConfigFileName holds the name of the file that the user uses to configure their project, not to be confused with InternalConfigFileName
const ConfigFileName = "activestate.yaml"

// InternalConfigNamespace holds the appdata folder name under which we store our config
const InternalConfigNamespace = "activestate"

// ConfigEnvVarName is the env var used to override the config dir that the state tool uses
const ConfigEnvVarName = "ACTIVESTATE_CLI_CONFIGDIR"

// DisableUpdates is the env var used to disable auto update
const DisableUpdates = "ACTIVESTATE_CLI_DISABLE_UPDATES"

// DisableRuntime is the env var used to disable downloading of runtimes, useful for CI or testing
const DisableRuntime = "ACTIVESTATE_CLI_DISABLE_RUNTIME"

// UpdateBranchEnvVarName is the env var that is used to override which branch to pull the update from
const UpdateBranchEnvVarName = "ACTIVESTATE_CLI_UPDATE_BRANCH"

// AutoUpdateTimeoutEnvVarName is the env var that is used to override the timeout for auto update checks
const AutoUpdateTimeoutEnvVarName = "ACTIVESTATE_CLI_AUTO_UPDATE_TIMEOUT"

// InternalConfigFileName is effectively the same as InternalConfigName, but includes our preferred extension
const InternalConfigFileName = "config.yaml"

// EnvironmentEnvVarName is the name of the environment variable that specifies the current environment (dev, qa, prod, etc.)
const EnvironmentEnvVarName = "ACTIVESTATE_ENVIRONMENT"

// ProjectEnvVarName is the name of the environment variable that specifies the path of the activestate.yaml config file.
const ProjectEnvVarName = "ACTIVESTATE_PROJECT"

// ActivatedStateEnvVarName is the name of the environment variable that is set when in an activated state, its value will be the path of the project
const ActivatedStateEnvVarName = "ACTIVESTATE_ACTIVATED"

// ForwardedStateEnvVarName is the name of the environment variable that is set when in an activated state, its value will be the path of the project
const ForwardedStateEnvVarName = "ACTIVESTATE_FORWARDED"

// APIUpdateURL is the URL for our update server
const APIUpdateURL = "https://s3.ca-central-1.amazonaws.com/cli-update/update/"

// APIArtifactURL is the URL for downloading artifacts
const APIArtifactURL = "https://s3.ca-central-1.amazonaws.com/cli-artifacts/"

// ArtifactFile is the name of the artifact json file contained within artifacts
const ArtifactFile = "artifact.json"

// UpdateStorageDir is the directory where updates will be stored
const UpdateStorageDir = "update/"

// DefaultNamespaceDomain is the domain used when no namespace is given and one has to be constructed
const DefaultNamespaceDomain = "github.com"

// AnalyticsTrackingID is our Google Analytics tracking ID
const AnalyticsTrackingID = "UA-118120158-1"

// APITokenNamePrefix is the name we give our api token
const APITokenNamePrefix = "activestate-platform-cli"

// KeypairLocalFileName is the name of the file (sans extension) that will hold the user's unencrypted
// private key in their config dir.
const KeypairLocalFileName = "private"

// DefaultRSABitLength represents the default RSA bit-length that will be assumed when
// generating new Keypairs.
const DefaultRSABitLength int = 4096

// ExpanderMaxDepth defines the maximum depth to fully expand a given value.
const ExpanderMaxDepth = int(10)

// StableBranch is the branch mapped to stable builds
const StableBranch = "stable"

// UnstableBranch is the branch used for unstable builds
const UnstableBranch = "unstable"

// ExperimentalBranch is the branch used for experimental builds
const ExperimentalBranch = "master"

// MonoAPIPath is the api path used for the platform api
const MonoAPIPath = "/api/v1"

// MonoURLProd is the host used for platform api calls when on production
const MonoURLProd = "https://platform.activestate.com" + MonoAPIPath

// MonoURLStage is the host used for platform api calls when on staging
const MonoURLStage = "https://staging.activestate.build" + MonoAPIPath

// MonoURLDev is the host used for platform api calls when on staging
const MonoURLDev = MonoURLStage

// SecretsAPIPath is the api path used for the secrets api
const SecretsAPIPath = "/api/secrets/v1"

// SecretsURLProd is the host used for secrets api calls when on production
const SecretsURLProd = "https://platform.activestate.com" + SecretsAPIPath

// SecretsURLStage is the host used for secrets api calls when on staging
const SecretsURLStage = "https://staging.activestate.build" + SecretsAPIPath

// SecretsURLDev is the host used for secrets api calls when on dev
const SecretsURLDev = "http://localhost:8080" + SecretsAPIPath

// ActivePythonDistsDir represents the base name of a directory where ActivePython dists will be installed under.
const ActivePythonDistsDir = "python"

// RuntimeInstallDirs represents the directory within a distribution archive where the distribution exists.
const RuntimeInstallDirs = "INSTALLDIR,perl"

// RuntimeMetaFile is the json file that holds meta information about our runtime
const RuntimeMetaFile = "support/metadata.json"

// HeadChefAPIPath is the api path used for the secrets api
const HeadChefAPIPath = "/sv/head-chef/"

// HeadChefURLProd is the host used for platform api calls when on production
const HeadChefURLProd = "wss://platform.activestate.com" + HeadChefAPIPath

// HeadChefURLStage is the host used for platform api calls when on staging
const HeadChefURLStage = "wss://staging.activestate.build" + HeadChefAPIPath

// HeadChefURLDev is the host used for platform api calls when on staging
const HeadChefURLDev = HeadChefURLStage

// HeadChefOrigin is the Origin header to use when making head-chef requests
const HeadChefOrigin = "https://localhost"

// InventoryAPIPath is the api path used for the secrets api
const InventoryAPIPath = "/sv/inventory-api"

// InventoryURLProd is the host used for platform api calls when on production
const InventoryURLProd = "https://platform.activestate.com" + InventoryAPIPath

// InventoryURLStage is the host used for platform api calls when on staging
const InventoryURLStage = "https://staging.activestate.build" + InventoryAPIPath

// InventoryURLDev is the host used for platform api calls when on staging
const InventoryURLDev = InventoryURLStage

// DeprecationInfoURL is the URL we check against to see what versions are deprecated
const DeprecationInfoURL = "https://s3.ca-central-1.amazonaws.com/cli-update/deprecation.json"

// DateFormatUser is the date format we use when communicating with the end-user
const DateFormatUser = "January 02, 2006"

// PlatformSignupURL is the account creation url used by the platform
const PlatformSignupURL = "https://platform.activestate.com/create-account"

// BugTrackerURL is the URL of our bug tracker
const BugTrackerURL = "https://github.com/ActiveState/state-tool/issues"

// UserAgentTemplate is the template used to generate the actual user agent, which includes runtime information as well as build information
const UserAgentTemplate = "{{.UserAgent}} ({{.OS}}; {{.OSVersion}}; {{.Architecture}})"
