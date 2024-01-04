package constants

import (
	"time"
)

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

// ServiceCommandName holds the name of our service command
const ServiceCommandName = "state-svc"

// ConfigFileName holds the name of the file that the user uses to configure their project, not to be confused with InternalConfigFileNameLegacy
const ConfigFileName = "activestate.yaml"

// ProjectConfigDirName is the name of the directory that holds project-specific data like commit ID.
// This folder does not hold ConfigFileName. It is a sibling to that file in a given directory.
const ProjectConfigDirName = ".activestate"

// BuildScriptFileName holds the name of the file that represents the build script used to generate the runtime
const BuildScriptFileName = "buildscript.yaml"

// CommitIdFileName is the name of the file in ProjectConfigDirName that contains a project's commit ID.
const CommitIdFileName = "commit"

// InternalConfigNamespace holds the appdata folder name under which we store our config
const InternalConfigNamespace = "activestate"

// HomeEnvVarName is the fallback env var used to determine the user's home directory.
const HomeEnvVarName = "ACTIVESTATE_HOME"

// ConfigEnvVarName is the env var used to override the config dir that the State Tool uses
const ConfigEnvVarName = "ACTIVESTATE_CLI_CONFIGDIR"

// CacheEnvVarName is the env var used to override the cache dir that the State Tool uses
const CacheEnvVarName = "ACTIVESTATE_CLI_CACHEDIR"

// LogEnvVarName is the env var used to override the log file path
const LogEnvVarName = "ACTIVESTATE_CLI_LOGFILE"

// LogBuildVerboseEnvVarName is the env var used to enable verbose build logging
const LogBuildVerboseEnvVarName = "ACTIVESTATE_CLI_BUILD_VERBOSE"

// DisableRuntime is the env var used to disable downloading of runtimes, useful for CI or testing
const DisableRuntime = "ACTIVESTATE_CLI_DISABLE_RUNTIME"

// DisableUpdates is the env var used to disable automatic updates
const DisableUpdates = "ACTIVESTATE_CLI_DISABLE_UPDATES"

// DisableLanguageTemplates is the env var used to disable templating for new activestate.yaml files
const DisableLanguageTemplates = "ACTIVESTATE_CLI_DISABLE_LANGUAGE_TEMPLATES"

// DisableProjectMigrationPrompt is the env var used to disable the project migration prompt for legacy projects.
// This is set by default for integration tests for backward-compatibility with old integration tests.
const DisableProjectMigrationPrompt = "ACTIVESTATE_CLI_DISABLE_PROJECT_MIGRATION_PROMPT"

// UpdateChannelEnvVarName is the env var that is used to override which channel to pull the update from
const UpdateChannelEnvVarName = "ACTIVESTATE_CLI_UPDATE_CHANNEL"

// InstallBuildDependencies is the env var that is used to override whether to install build dependencies
const InstallBuildDependencies = "ACTIVESTATE_CLI_INSTALL_BUILD_DEPENDENCIES"

// InternalConfigFileNameLegacy is effectively the same as InternalConfigName, but includes our preferred extension
const InternalConfigFileNameLegacy = "config.yaml"

// InternalConfigFileName is the filename used for our sqlite based settings db
const InternalConfigFileName = "config.db"

// AutoUpdateTimeoutEnvVarName is the name of the environment variable that can be set to override the allowed timeout to check for an available auto-update
const AutoUpdateTimeoutEnvVarName = "ACTIVESTATE_CLI_UPDATE_TIMEOUT"

// EnvironmentEnvVarName is the name of the environment variable that specifies the current environment (dev, qa, prod, etc.)
const EnvironmentEnvVarName = "ACTIVESTATE_ENVIRONMENT"

// ProjectEnvVarName is the name of the environment variable that specifies the path of the activestate.yaml config file.
const ProjectEnvVarName = "ACTIVESTATE_PROJECT"

// ActivatedStateEnvVarName is the name of the environment variable that is set when in an activated state, its value will be the path of the project
const ActivatedStateEnvVarName = "ACTIVESTATE_ACTIVATED"

// ActivatedStateIDEnvVarName is the name of the environment variable that is set when in an activated state, its value will be a unique id identifying a specific instance of an activated state
const ActivatedStateIDEnvVarName = "ACTIVESTATE_ACTIVATED_ID"

// ActivatedStateNamespaceEnvVarName is the name of the environment variable that specifies the activated state's org/project namespace.
const ActivatedStateNamespaceEnvVarName = "ACTIVESTATE_ACTIVATED_NAMESPACE"

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

// ProfileEnvVarName is the name of the environment variable that specifies whether profiling should be run.
const ProfileEnvVarName = "ACTIVESTATE_PROFILE"

// SessionTokenEnvVarName records the session token
const SessionTokenEnvVarName = "ACTIVESTATE_SESSION_TOKEN"

// OverrideSessionTokenEnvVarName overrides SessionTokenEnvVarName for integration tests.
const OverrideSessionTokenEnvVarName = "ACTIVESTATE_OVERRIDE_SESSION_TOKEN"

// UpdateTagEnvVarName
const UpdateTagEnvVarName = "ACTIVESTATE_UPDATE_TAG"

// NonInteractiveEnvVarName is the name of the environment variable that specifies whether to run the State Tool without prompts
const NonInteractiveEnvVarName = "ACTIVESTATE_NONINTERACTIVE"

// E2ETestEnvVarName is the name of the environment variable that specifies that we are running under E2E tests
const E2ETestEnvVarName = "ACTIVESTATE_E2E_TEST"

// HeartbeatIntervalEnvVarName is the name of the environment variable used to override the heartbeat interval
const HeartbeatIntervalEnvVarName = "ACTIVESTATE_HEARTBEAT_INTERVAL"

// OverwriteDefaultInstallationPathEnvVarName is the environment variable name to overwrite the default installation path FOR TESTING PURPOSES ONLY
const OverwriteDefaultInstallationPathEnvVarName = "ACTIVESTATE_TEST_INSTALL_PATH"

// OverwriteDefaultSystemPathEnvVarName is the environment variable name to overwrite the system app installation directory updates FOR TESTING PURPOSES ONLY
const OverwriteDefaultSystemPathEnvVarName = "ACTIVESTATE_TEST_SYSTEM_PATH"

// OverrideOSNameEnvVarName is used to override the OS name used when initializing projects
const OverrideOSNameEnvVarName = "ACTIVESTATE_OVERRIDE_OS_NAME"

// TestAutoUpdateEnvVarName is used to test auto updates, when set to true will always attempt to auto update
const TestAutoUpdateEnvVarName = "ACTIVESTATE_TEST_AUTO_UPDATE"

// ForceUpdateEnvVarName is used to force state tool to update, regardless of whether the update is equal to the current version
const ForceUpdateEnvVarName = "ACTIVESTATE_FORCE_UPDATE"

// ShimEnvVarName is used to instruct State Tool that it's being executed as part of a shim
const ShimEnvVarName = "ACTIVESTATE_SHIM"

// AnalyticsLogEnvVarName is used to instruct State Tool to report analytics events to the given file
const AnalyticsLogEnvVarName = "ACTIVESTATE_ANALYTICS_LOG"

// DisableAnalyticsEnvVarName is used to instruct State Tool to not send data to Google Analytics.
const DisableAnalyticsEnvVarName = "ACTIVESTATE_CLI_DISABLE_ANALYTICS"

// OptinUnstableEnvVarName is used to instruct State Tool to opt-in to unstable features
const OptinUnstableEnvVarName = "ACTIVESTATE_OPTIN_UNSTABLE"

// ServiceSockDir overrides the default socket path root diriectory used by the state service
const ServiceSockDir = "ACTIVESTATE_SVC_SOCK"

// MessagesOverrideEnvVarName is used to override the location of the messages file (for testing purposes - should hold local filepath)
const MessagesOverrideEnvVarName = "ACTIVESTATE_MESSAGES_OVERRIDE"

// DisableErrorTipsEnvVarName disables the display of tips in error messages.
// This should only be used by the installer so-as not to pollute error message output.
const DisableErrorTipsEnvVarName = "ACTIVESTATE_CLI_DISABLE_ERROR_TIPS"

// DebugServiceRequestsEnvVarName is used to instruct State Tool to turn on debug logging of service requests
const DebugServiceRequestsEnvVarName = "ACTIVESTATE_DEBUG_SERVICE_REQUESTS"

// InstallPathOverrideEnvVarName is used to override the default installation path of the state tool.
// This is intended for use in our integration tests, not by end-users.
const InstallPathOverrideEnvVarName = "ACTIVESTATE_CLI_INSTALLPATH_OVERRIDE"

// AutostartPathOverrideEnvVarName is used to override the default autostart path of the state service.
const AutostartPathOverrideEnvVarName = "ACTIVESTATE_CLI_AUTOSTARTPATH_OVERRIDE"

// AppInstallDirOverrideEnvVarName is used to override the default app installation path of the state tool.
const AppInstallDirOverrideEnvVarName = "ACTIVESTATE_CLI_APPINSTALLDIR_OVERRIDE"

// SvcAuthPollingRateEnvVarName is used to override the default polling rate for syncing the authenticated state with the svc
const SvcAuthPollingRateEnvVarName = "ACTIVESTATE_SVC_AUTH_POLLING_RATE"

// StateSvcLogRotateInvervalEnvVarName is the environment variable used to override the default
// log rotation timer interval (1 minute).
const SvcLogRotateIntervalEnvVarName = "ACTIVESTATE_CLI_LOG_ROTATE_INTERVAL_MS"

// APIUpdateInfoURL is the URL for our update info server
const APIUpdateInfoURL = "https://platform.activestate.com/sv/state-update/api/v1"

// APIUpdateURL is the URL for our update files
const APIUpdateURL = "https://state-tool.s3.amazonaws.com/update/state"

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

// ArtifactCacheSizeEnvVarName is the maximum size in MB of the artifact cache.
// The default is 500MB.
const ArtifactCacheSizeEnvVarName = "ACTIVESTATE_ARTIFACT_CACHE_SIZE_MB"

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

// ReleaseChannel is the channel used for release builds
const ReleaseChannel = "release"

// BetaChannel is the channel used for beta builds
const BetaChannel = "beta"

// ExperimentalChannel is the channel used for experimental builds
const ExperimentalChannel = "master"

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

// BuildplanAPIPath is the path used for the build planner api
const BuildplanAPIPath = "/sv/buildplanner/graphql"

// RequirementsImportAPIPath is the path used for the requirements import api
const RequirementsImportAPIPath = "/sv/reqsvc/reqs"

// BuildPlannerAPIPath is the path used for the build planner api
const BuildPlannerAPIPath = "/sv/buildplanner/graphql"

// MessagesInfoURL is the URL we check against to see what versions are deprecated
const MessagesInfoURL = "https://state-tool.s3.amazonaws.com/messages.json"

// DateFormatUser is the date format we use when communicating with the end-user
const DateFormatUser = "January 02, 2006"

// DateTimeFormatUser is the datetime format we use when communicating with the end-user
const DateTimeFormatUser = "2 Jan 2006 15:04"

// DateTimeFormatRecord is the datetime format we use when recording for internal use
const DateTimeFormatRecord = "Mon Jan 2 2006 15:04:05 -0700 MST"

// PlatformSignupPath is the account creation path used by the platform
const PlatformSignupPath = "/create-account"

// DocumentationURL is the url for the state tool documentation
const DocumentationURL = "http://docs.activestate.com/platform/state/"

// DocumentationURLGetStarted is the documentation URL for creating projects
const DocumentationURLGetStarted = DocumentationURL + "create-project/?utm_source=platform-application-gui&utm_medium=activestate-desktop&utm_content=drop-down&utm_campaign=maru"

// DocumentationURLMismatch is the documentation URL for the project mismatch warning
const DocumentationURLMismatch = DocumentationURL + "troubleshooting/git-project-mismatch/"

// DocumentationURLLocking is the documentation URL for locking
const DocumentationURLLocking = DocumentationURL + "advanced-topics/locking/"

// ActiveStateBlogURL is the URL for the ActiveState Blog
const ActiveStateBlogURL = "https://www.activestate.com/blog/?utm_source=platform-application-gui&utm_medium=activestate-desktop&utm_content=drop-down&utm_campaign=maru"

// ActiveStateSupportURL is the URL for the AciveState support page
const ActiveStateSupportURL = "https://www.activestate.com/support/?utm_source=platform-application-gui&utm_medium=activestate-desktop&utm_content=drop-down&utm_campaign=maru"

// ActiveStateDashboardURL is the URL for the ActiveState account preferences page
const ActiveStateDashboardURL = "https://platform.activestate.com/?utm_source=platform-application-gui&utm_medium=activestate-desktop&utm_content=drop-down&utm_campaign=maru"

// BugTrackerURL is the URL of our bug tracker
const BugTrackerURL = "https://github.com/ActiveState/state-tool/issues"

// UserAgentTemplate is the template used to generate the actual user agent, which includes runtime information as well as build information
const UserAgentTemplate = "{{.UserAgent}} ({{.OS}}; {{.OSVersion}}; {{.Architecture}})"

// PlatformURL is the base domain for the production platform
const PlatformURL = "platform.activestate.com"

// CheatSheetURL is the URL for the State Tool Cheat Sheet
const CheatSheetURL = "https://platform.activestate.com/state-tool-cheat-sheet"

// StateToolRollbarToken is the token used by the State Tool to talk to rollbar
const StateToolRollbarToken = "4062b6b437ed40e9aa710ce8931d7897"

// StateServiceRollbarToken is the token used by the State Service to talk to rollbar
const StateServiceRollbarToken = "9dce777154b84824b1a16eb4654886a9"

// StateInstallerRollbarToken is the token used by the State Installer to talk to rollbar
// Todo It is currently the same as the State Tool's
const StateInstallerRollbarToken = "f4105fee70c5478eab063abb1acdffa9"

// OfflineInstallerRollbarToken is the token used by the Offline Installer to talk to rollbar
const OfflineInstallerRollbarToken = "0ab5e19218bd494680bf8f5d08cf37ad"

// {OS}Bit{Depth}UUID constants are the UUIDs associated with the relevant OSes
// in the platform DB and leveraged by the Dashboard (PlatformMapping).
const (
	Win10Bit64UUID = "78977bc8-0f32-519d-80f3-9043f059398c"
	LinuxBit64UUID = "7c998ec2-7491-4e75-be4d-8885800ef5f2"
	MacBit64UUID   = "46a5b48f-226a-4696-9746-ba4d50d661c2"
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

// RuntimeBuildPlanStore containts a serialization of the build plan used to create this build
const RuntimeBuildPlanStore = "build_plan"

// BuildExpressionStore holds the cached build expression for the current commit ID.
const BuildExpressionStore = "build_expression"

// StateToolMarketingPage links to the marketing page for the state tool
const StateToolMarketingPage = "https://www.activestate.com/products/platform/state-tool/"

// PlatformMarketingPage links to the marketing page for the ActiveState Platform
const PlatformMarketingPage = "https://www.activestate.com/products/platform/"

// TermsOfServiceURLLatest is the URL to get the latest terms of service in PDF form
const TermsOfServiceURLLatest = "https://www.activestate.com/wp-content/uploads/2018/10/activestate_platform_terms_service_agreement.pdf"

// RCAppendDeployStartLine is the start line used to denote our deploy environment config in RC files
const RCAppendDeployStartLine = "-- START ACTIVESTATE DEPLOY RUNTIME ENVIRONMENT"

// RCAppendDeployStopLine is the end line used to denote our deploy environment config in RC files
const RCAppendDeployStopLine = "-- STOP ACTIVESTATE DEPLOY RUNTIME ENVIRONMENT"

// RCAppendOfflineInstallStartLine is the start line used to denote our deploy environment config in RC files
const RCAppendOfflineInstallStartLine = "-- START ACTIVESTATE OFFLINE INSTALL RUNTIME ENVIRONMENT"

// RCAppendOfflineInstallStopLine is the end line used to denote our deploy environment config in RC files
const RCAppendOfflineInstallStopLine = "-- STOP ACTIVESTATE OFFLINE INSTALL RUNTIME ENVIRONMENT"

// RCAppendDefaultStartLine is the start line used to denote our default environment config in RC files
const RCAppendDefaultStartLine = "-- START ACTIVESTATE DEFAULT RUNTIME ENVIRONMENT"

// RCAppendDefaultStopLine is the end line used to denote our default environment config in RC files
const RCAppendDefaultStopLine = "-- STOP ACTIVESTATE DEFAULT RUNTIME ENVIRONMENT"

// RCAppendInstallStartLine is the start line used to denote our default installation config in RC files
const RCAppendInstallStartLine = "-- START ACTIVESTATE INSTALLATION"

// RCAppendInstallStopLine is the end line used to denote our default installation config in RC files
const RCAppendInstallStopLine = "-- STOP ACTIVESTATE INSTALLATION"

// RCAppendAutostartStartLine is the start line used to denote our autostart executables in RC files
const RCAppendAutostartStartLine = "## START ACTIVESTATE AUTOSTART"

// RCAppendAutostartStartLine is the end line used to denote our autostart executables in RC files
const RCAppendAutostartStopLine = "## STOP ACTIVESTATE AUTOSTART"

// ForumsURL is the URL to the state tool forums
const ForumsURL = "https://community.activestate.com/c/state-tool/"

// GlobalDefaultPrefname is the pref that holds the path to the globally defaulted project
const GlobalDefaultPrefname = "projects.active.path"

// DefaultBranchName is the default branch name used on platform projects
const DefaultBranchName = "main"

// UnstableConfig is the config key used to determine whether the user has opted in to unstable commands
const UnstableConfig = "optin.unstable"

// ReportErrorsConfig is the config key used to determine if we will send rollbar reports
const ReportErrorsConfig = "report.errors"

// ReportAnalyticsConfig is the config key used to determine if we will send analytics reports
const ReportAnalyticsConfig = "report.analytics"

// SvcAppName is the name we give our state-svc application
const SvcAppName = "State Service"

// SvcLaunchFileName is the name we give files pertaining to our state-svc application
const SvcLaunchFileName = "state-svc"

// StateAppName is the name we give our state cli executable
const StateAppName = "State Tool"

// StateSvcCmd is the name of the state-svc binary
const StateSvcCmd = "state-svc"

// AutostartSvcConfigKey is the config key used to determine if the service should be run on startup.
const AutostartSvcConfigKey = "autostart.svc"

// StateCmd is the name of the state tool binary
const StateCmd = "state"

// StateInstallerCmd is the name of the state installer binary
const StateInstallerCmd = "state-installer"

// StateRemoteInstallerCmd is the name of the state remote installer binary
const StateRemoteInstallerCmd = "state-remote-installer"

// InstallerName is the name we give to our state-installer executable
const InstallerName = "State Installer"

// StateExecutorCmd is the name of the state executor binary
const StateExecutorCmd = "state-exec"

// ToplevelInstallArchiveDir is the top-level directory for files in an installation archive
const ToplevelInstallArchiveDir = "state-install"

// FirstMultiFileStateToolVersion is the State Tool version that introduced multi-file updates
const FirstMultiFileStateToolVersion = "0.29.0"

// ExecRecursionLevelEnvVarName is an environment variable storing the number of times the executor has been called recursively
const ExecRecursionLevelEnvVarName = "ACTIVESTATE_CLI_EXECUTOR_RECURSION_LEVEL"

// ExecRecursionEnvVarName is an environment variable storing a string representation of the current recursion
const ExecRecursionEnvVarName = "ACTIVESTATE_CLI_EXECUTOR_RECURSION"

// ExecRecursionAllowEnvVarName is an environment variable overriding the recursion allowance
const ExecRecursionAllowEnvVarName = "ACTIVESTATE_CLI_EXECUTOR_RECURSION_ALLOW"

// ExecRecursionMaxLevelEnvVarName is an environment variable storing the number of times the executor may be called recursively
const ExecRecursionMaxLevelEnvVarName = "ACTIVESTATE_CLI_EXECUTOR_MAX_RECURSION_LEVEL"

// InstallerNoSubshell is an environment variable that if set to true will tell the installer not to start a subshell
const InstallerNoSubshell = "ACTIVESTATE_CLI_INSTALLER_NO_SUBSHELL"

// InstallSourceFile is the file we use to record what installed the state tool
const InstallSourceFile = "installsource.txt"

// PpmShim is the name of the ppm shim
const PpmShim = "ppm"

// PipShim is the name of the pip shim
const PipShim = "pip"

// AutoUpdateConfigKey is the config key for storing whether or not autoupdates can be performed
const AutoUpdateConfigKey = "autoupdate"

// PreservePs1ConfigKey is the config key that specifies whether to modify the shell PS1/prompt to show [org/project] info.
const PreservePs1ConfigKey = "preserve.prompt"

// DefaultAnalyticsPixel is the default url for the analytics pixel
const DefaultAnalyticsPixel = "https://state-tool.s3.amazonaws.com/pixel"

// AnalyticsPixelOverrideEnv is the environment variable to check for overriding the analytics pixel url
const AnalyticsPixelOverrideEnv = "ACTIVESTATE_CLI_ANALYTICS_PIXEL"

// TerminalAnimationInterval is the interval we use for terminal animations
const TerminalAnimationInterval = 150 * time.Millisecond

// RuntimeSetupWaitEnvVarName is only used for an integration test to pause installation and wait
// for Ctrl+C.
const RuntimeSetupWaitEnvVarName = "ACTIVESTATE_CLI_RUNTIME_SETUP_WAIT"

// PlatformApiRequestRequestsEnvVarName is only used for an integration test to print some Platform
// API request info.
const PlatformApiPrintRequestsEnvVarName = "ACTIVESTATE_CLI_PLATFORM_API_PRINT_REQUESTS"

// ActiveStateCIEnvVarName is the environment variable set when running in an ActiveState CI environment.
const ActiveStateCIEnvVarName = "ACTIVESTATE_CI"

// OverrideSandbox is the environment variable to set when overriding the sandbox for integration tests.
const OverrideSandbox = "ACTIVESTATE_TEST_OVERRIDE_SANDBOX"
