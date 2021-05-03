package constants

// LibraryName contains the main name of this library
const LibraryName = "cli"

// LibraryOwner contains the name of the owner of this library
const LibraryOwner = "ActiveState"

// LibraryNamespace is the namespace that the library belongs to
const LibraryNamespace = "github.com/ActiveState/"

// LibraryLicense is the license that the library is distributed under.
const LibraryLicense = "BSD 3"

// CommandName holds the name of our command
const CommandName = "state"

// ConfigFileName holds the name of the file that the user uses to configure their project, not to be confused with InternalConfigFileName
const ConfigFileName = "activestate.yaml"

// InternalConfigNamespace holds the appdata folder name under which we store our config
const InternalConfigNamespace = "activestate"

// ConfigEnvVarName is the env var used to override the config dir that the State Tool uses
const ConfigEnvVarName = "ACTIVESTATE_CLI_CONFIGDIR"

// CacheEnvVarName is the env var used to override the cache dir that the State Tool uses
const CacheEnvVarName = "ACTIVESTATE_CLI_CACHEDIR"

// ExecEnvVarName is the env var used to find out if we are shimming recursively
const ExecEnvVarName = "ACTIVESTATE_CLI_SHIMMED_COMMAND"

// DisableUpdates is the env var used to disable auto update
const DisableUpdates = "ACTIVESTATE_CLI_DISABLE_UPDATES"

// DisableRuntime is the env var used to disable downloading of runtimes, useful for CI or testing
const DisableRuntime = "ACTIVESTATE_CLI_DISABLE_RUNTIME"

// UpdateBranchEnvVarName is the env var that is used to override which branch to pull the update from
const UpdateBranchEnvVarName = "ACTIVESTATE_CLI_UPDATE_BRANCH"

// UpdateHailFileName is the file name used to pass messages from sub-processes to the parent.
const UpdateHailFileName = "hail-update"

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

// ActivatedStateIDEnvVarName is the name of the environment variable that is set when in an activated state, its value will be a unique id identifying a specific instance of an activated state
const ActivatedStateIDEnvVarName = "ACTIVESTATE_ACTIVATED_ID"

// ForwardedStateEnvVarName is the name of the environment variable that is set when in an activated state, its value will be the path of the project
const ForwardedStateEnvVarName = "ACTIVESTATE_FORWARDED"

// PrivateKeyEnvVarName is the name of the environment variable that specifies the private key file to use for decrypting secrets (overriding user config).
const PrivateKeyEnvVarName = "ACTIVESTATE_PRIVATE_KEY"

// APIKeyEnvVarName is the name of the environment variable that specifies the API Key to use for api authentication (overriding user config).
const APIKeyEnvVarName = "ACTIVESTATE_API_KEY"

// APIHostEnvVarName is the name of the environment variable that specifies the API host, specifying this overrides the activestate.yaml api url config
const APIHostEnvVarName = "ACTIVESTATE_API_HOST"

// APIInsecureEnvVarName is the name of the environment variable that specifies whether the API hostURI should be insecure.
const APIInsecureEnvVarName = "ACTIVESTATE_API_INSECURE"

// CPUProfileEnvVarName is the name of the environment variable that specifies whether CPU profiling should be run.
const CPUProfileEnvVarName = "ACTIVESTATE_PROFILE_CPU"

// NonInteractive is the name of the environment variable that specifies whether to run the State Tool without prompts
const NonInteractive = "ACTIVESTATE_NONINTERACTIVE"

// APIUpdateURL is the URL for our update server
const APIUpdateURL = "https://state-tool.s3.amazonaws.com/update/"

// APIArtifactURL is the URL for downloading artifacts
const APIArtifactURL = "https://s3.ca-central-1.amazonaws.com/cli-artifacts/"

// ArtifactFile is the name of the artifact json file contained within artifacts
const ArtifactFile = "artifact.json"

// ArtifactArchiveName is the standardized name of an artifact archive
const ArtifactArchiveName = "artifact.tar.gz"

// ArtifactCacheFileName is the standardized name of an artifact cache file
const ArtifactCacheFileName = "artifact_cache.json"

// ArtifactMetaDir is the directory in which we store meta information about artifacts
const ArtifactMetaDir = "artifacts"

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

// ReleaseBranch is the branch used for release builds
const ReleaseBranch = "release"

// BetaBranch is the branch used for beta builds
const BetaBranch = "beta"

// ExperimentalBranch is the branch used for experimental builds
const ExperimentalBranch = "master"

// MonoAPIPath is the api path used for the platform api
const MonoAPIPath = "/api/v1"

// DefaultAPIHost is the host used for platform api calls when on production
const DefaultAPIHost = "platform.activestate.com"

// SecretsAPIPath is the api path used for the secrets api
const SecretsAPIPath = "/api/secrets/v1"

// SecretsURL is the host used for secrets api calls when on production
const SecretsURL = "https://platform.activestate.com" + SecretsAPIPath

// HeadChefAPIPath is the api path used for the headchef api
const HeadChefAPIPath = "/sv/head-chef"

// BuildLogStreamerPath is the websocket API used for streaming build results
const BuildLogStreamerPath = "/sv/build-log-streamer"

// InventoryAPIPath is the api path used for the secrets api
const InventoryAPIPath = "/sv/inventory-api-v1"

// GraphqlAPIPath is the path used for the platform graphql api
const GraphqlAPIPath = "/graphql/v1/graphql"

// MediatorAPIPath is the path used for the platform mediator api
const MediatorAPIPath = "/sv/mediator/api"

// RequirementsImportAPIPath is the path used for the requirements import api
const RequirementsImportAPIPath = "/sv/reqsvc/reqs"

// DeprecationInfoURL is the URL we check against to see what versions are deprecated
const DeprecationInfoURL = "https://state-tool.s3.amazonaws.com/deprecation.json"

// DateFormatUser is the date format we use when communicating with the end-user
const DateFormatUser = "January 02, 2006"

// DateTimeFormatUser is the datetime format we use when communicating with the end-user
const DateTimeFormatUser = "2 Jan 2006 15:04"

// DateTimeFormatRecord is the datetime format we use when recording for internal use
const DateTimeFormatRecord = "Mon Jan 2 2006 15:04:05 -0700 MST"

// PlatformSignupURL is the account creation url used by the platform
const PlatformSignupURL = "https://platform.activestate.com" + "/create-account"

// DocumentationURL is the url for the state tool documentation
const DocumentationURL = "http://docs.activestate.com/platform/state/"

// DocumentationURLHeadless is the documentation URL for headless state docs
const DocumentationURLHeadless = DocumentationURL + "advanced-topics/detached/"

// ActiveStateBlogURL is the URL for the ActiveState Blog
const ActiveStateBlogURL = "https://www.activestate.com/blog/"

// ActiveStateSupportURL is the URL for the AciveState support page
const ActiveStateSupportURL = "https://www.activestate.com/support/"

// BugTrackerURL is the URL of our bug tracker
const BugTrackerURL = "https://github.com/ActiveState/state-tool/issues"

// UserAgentTemplate is the template used to generate the actual user agent, which includes runtime information as well as build information
const UserAgentTemplate = "{{.UserAgent}} ({{.OS}}; {{.OSVersion}}; {{.Architecture}})"

// PlatformURL is the base domain for the production platform
const PlatformURL = "platform.activestate.com"

// StateToolRollbarToken is the token used by the State Tool to talk to rollbar
const StateToolRollbarToken = "cc836c27caf344f7befab5b707ed7d4e"

// StateTrayRollbarToken is the token used by the State Tray to talk to rollbar
const StateTrayRollbarToken = "d786a99eabf24617b82c44dfab19d907"

// StateServiceRollbarToken is the token used by the State Service to talk to rollbar
const StateServiceRollbarToken = "8591fd01f23a41acb14d478c85638d92"

// StateInstallerRollbarToken is the token used by the State Installer to talk to rollbar
// Todo It is currently the same as the State Tool's
const StateInstallerRollbarToken = "cc836c27caf344f7befab5b707ed7d4e"

// {OS}Bit{Depth}UUID constants are the UUIDs associated with the relevant OSes
// in the platform DB.
const (
	Win10Bit64UUID = "78977bc8-0f32-519d-80f3-9043f059398c"
	LinuxBit64UUID = "681d5381-518c-5f4c-b367-df05c8d525e2"
	MacBit64UUID   = "96b7e6f2-bebf-564c-bc1c-f04482398f38"
	ValidZeroUUID  = "00000000-0000-0000-0000-000000000000"
)

// ActivePythonDistsDir represents the base name of a directory where ActivePython dists will be installed under.
const ActivePythonDistsDir = "python"

// RuntimeInstallDirs represents the directory within a distribution archive where the distribution exists.
const RuntimeInstallDirs = "INSTALLDIR,perl"

// RuntimeMetaFile is the json file that holds meta information about our runtime
const RuntimeMetaFile = "metadata.json"

// RuntimeDefinitionFilename is the filename for runtime meta data bundled with artifacts, if they are built by the alternative builder
const RuntimeDefinitionFilename = "runtime.json"

// LocalRuntimeEnvironmentDirectory is the directory (relative to the installation of a runtime build) where runtime definition files are stored
const LocalRuntimeEnvironmentDirectory = "_runtime_store"

// LocalRuntimeTempDirectory is the directory (relative to the installation of a runtime build) where temp files are stored
const LocalRuntimeTempDirectory = "_runtime_temp"

// RuntimeInstallationCompleteMarker is created after all artifacts have been installed
// Check for existence of this file to ensure that the installation has not been interrupted prematurely.
const RuntimeInstallationCompleteMarker = "completed"

// RuntimeBuildEngineStore contains the name of the build engine that was used to create this runtime
const RuntimeBuildEngineStore = "build_engine"

// RuntimeRecipeStore contains a serialization of the recipe used to create this build
const RuntimeRecipeStore = "recipe"

// StateToolMarketingPage links to the marketing page for the state tool
const StateToolMarketingPage = "https://www.activestate.com/products/platform/state-tool/"

// PlatformMarketingPage links to the marketing page for the ActiveState Platform
const PlatformMarketingPage = "https://www.activestate.com/products/platform/"

// TermsOfServiceURLText is the URL to get the current terms of service in txt form
const TermsOfServiceURLText = "https://www.activestate.com/wp-content/uploads/2020/03/activestate_platform_terms_service_agreement.txt"

// TermsOfServiceURLLatest is the URL to get the latest terms of service in PDF form
const TermsOfServiceURLLatest = "https://www.activestate.com/wp-content/uploads/2018/10/activestate_platform_terms_service_agreement.pdf"

// RCAppendDeployStartLine is the start line used to denote our deploy environment config in RC files
const RCAppendDeployStartLine = "-- START ACTIVESTATE DEPLOY RUNTIME ENVIRONMENT"

// RCAppendDeployStopLine is the end line used to denote our deploy environment config in RC files
const RCAppendDeployStopLine = "-- STOP ACTIVESTATE DEPLOY RUNTIME ENVIRONMENT"

// RCAppendDefaultStartLine is the start line used to denote our default environment config in RC files
const RCAppendDefaultStartLine = "-- START ACTIVESTATE DEFAULT RUNTIME ENVIRONMENT"

// RCAppendDefaultStopLine is the end line used to denote our default environment config in RC files
const RCAppendDefaultStopLine = "-- STOP ACTIVESTATE DEFAULT RUNTIME ENVIRONMENT"

// RCAppendInstallStartLine is the start line used to denote our default environment config in RC files
const RCAppendInstallStartLine = "-- START ACTIVESTATE INSTALLATION"

// RCAppendInstallStopLine is the end line used to denote our default environment config in RC files
const RCAppendInstallStopLine = "-- STOP ACTIVESTATE INSTALLATION"

// ForumsURL is the URL to the state tool forums
const ForumsURL = "https://community.activestate.com/c/state-tool/"

// GlobalDefaultPrefname is the pref that holds the path to the globally defaulted project
const GlobalDefaultPrefname = "default_project_path"

// DefaultBranchName is the default branch name used on platform projects
const DefaultBranchName = "main"

// SvcConfigPort is the config key used for storing the svc port
const SvcConfigPort = "svc-port"

// SvcConfigPid is the config key used for storing the svc pid
const SvcConfigPid = "svc-pid"

// TrayAppName is the name we give our systray application
const TrayAppName = "ActiveState Desktop"

// SvcAppName is the name we give our state-svc application
const SvcAppName = "State Service"

// StateAppName is the name we give our state cli executable
const StateAppName = "State Tool"

// ToplevelInstallArchiveDir is the top-level directory for files in an installation archive
// Cf., https://www.pivotaltracker.com/story/show/177781411
const ToplevelInstallArchiveDir = "state-install"

// FirstMultiFileStateToolVersion is the State Tool version that introduced multi-file updates
const FirstMultiFileStateToolVersion = "0.29.0"
