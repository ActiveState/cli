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
const APIUpdateURL = "https://s3.ca-central-1.amazonaws.com/cli-update/update/"

// APIArtifactURL is the URL for downloading artifacts
const APIArtifactURL = "https://s3.ca-central-1.amazonaws.com/cli-artifacts/"

// ArtifactFile is the name of the artifact json file contained within artifacts
const ArtifactFile = "artifact.json"

// ArtifactArchiveName is the standardized name of an artifact archive
const ArtifactArchiveName = "artifact.tar.gz"

// ArtifactCacheFileName is the standardized name of an artifact cache file
const ArtifactCacheFileName = "artifact_cache.json"

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

// DefaultAPIHost is the host used for platform api calls when on production
const DefaultAPIHost = "platform.activestate.com"

// SecretsAPIPath is the api path used for the secrets api
const SecretsAPIPath = "/api/secrets/v1"

// SecretsURL is the host used for secrets api calls when on production
const SecretsURL = "https://platform.activestate.com" + SecretsAPIPath

// HeadChefAPIPath is the api path used for the headchef api
const HeadChefAPIPath = "/sv/head-chef"

// InventoryAPIPath is the api path used for the secrets api
const InventoryAPIPath = "/sv/inventory-api-v1"

// GraphqlAPIPath is the path used for the platform graphql api
const GraphqlAPIPath = "/graphql/v1/graphql"

// RequirementsImportAPIPath is the path used for the requiremments import api
const RequirementsImportAPIPath = "/sv/reqsvc/reqs"

// DeprecationInfoURL is the URL we check against to see what versions are deprecated
const DeprecationInfoURL = "https://s3.ca-central-1.amazonaws.com/cli-update/deprecation.json"

// DateFormatUser is the date format we use when communicating with the end-user
const DateFormatUser = "January 02, 2006"

// DateTimeFormatUser is the datetime format we use when communicating with the end-user
const DateTimeFormatUser = "2 Jan 2006 15:04"

// PlatformSignupURL is the account creation url used by the platform
const PlatformSignupURL = "https://platform.activestate.com" + "/create-account"

// DocumentationURL is the url for the state tool documentation
const DocumentationURL = "http://docs.activestate.com/platform/state/"

// BugTrackerURL is the URL of our bug tracker
const BugTrackerURL = "https://github.com/ActiveState/state-tool/issues"

// UserAgentTemplate is the template used to generate the actual user agent, which includes runtime information as well as build information
const UserAgentTemplate = "{{.UserAgent}} ({{.OS}}; {{.OSVersion}}; {{.Architecture}})"

// PlatformURL is the base domain for the production platform
const PlatformURL = "platform.activestate.com"

// RollbarToken is the token used to talk to rollbar
const RollbarToken = "cc836c27caf344f7befab5b707ed7d4e"

// {OS}Bit{Depth}UUID constants are the UUIDs associated with the relevant OSes
// in the platform DB.
const (
	Win10Bit64UUID = "78977bc8-0f32-519d-80f3-9043f059398c"
	LinuxBit64UUID = "681d5381-518c-5f4c-b367-df05c8d525e2"
	MacBit64UUID   = "96b7e6f2-bebf-564c-bc1c-f04482398f38"
)

// ActivePythonDistsDir represents the base name of a directory where ActivePython dists will be installed under.
const ActivePythonDistsDir = "python"

// RuntimeInstallDirs represents the directory within a distribution archive where the distribution exists.
const RuntimeInstallDirs = "INSTALLDIR,perl"

// RuntimeMetaFile is the json file that holds meta information about our runtime
const RuntimeMetaFile = "support/metadata.json"

// RuntimeDefinitionFilename is the filename for runtime meta data bundled with artifacts, if they are built by the alternative builder
const RuntimeDefinitionFilename = "runtime.json"

// LocalRuntimeEnvironmentDirectory is the directory (relative to the installation of a runtime build) where runtime definition files are stored
const LocalRuntimeEnvironmentDirectory = "_runtime_env"

// RuntimeInstallationCompleteMarker is created after all artifacts have been installed
// Check for existence of this file to ensure that the installation has not been interrupted prematurely.
const RuntimeInstallationCompleteMarker = "support/completed"

// TermsOfServiceURLText is the URL to get the current terms of service in txt form
const TermsOfServiceURLText = "https://www.activestate.com/wp-content/uploads/2020/03/activestate_platform_terms_service_agreement.txt"

// TermsOfServiceURLLatest is the URL to get the latest terms of service in PDF form
const TermsOfServiceURLLatest = "https://www.activestate.com/wp-content/uploads/2018/10/activestate_platform_terms_service_agreement.pdf"

// RCAppendStartLine is the start line used to denote our environment config in RC files
const RCAppendStartLine = "-- START ACTIVESTATE RUNTIME ENVIRONMENT"

// RCAppendEndLine is the end line used to denote our environment config in RC files
const RCAppendStopLine = "-- STOP ACTIVESTATE RUNTIME ENVIRONMENT"

// ForumsURL is the URL to the state tool forums
const ForumsURL = "https://community.activestate.com/c/state-tool/9"
