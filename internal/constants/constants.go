package constants

// VersionNumber holds the current version of our cli
const VersionNumber = "0.1.1"

// LibraryName contains the main name of this library
const LibraryName = "cli"

// LibraryNamespace is the namespace that the library belongs to
const LibraryNamespace = "github.com/ActiveState/"

// CommandName holds the name of our command
const CommandName = "state"

// ConfigNamespace holds the appdata folder name under which we store our config
const ConfigNamespace = "activestate"

// ConfigName is used to inform viper and our config lib about the name of the config file
const ConfigName = "activestate"

// ConfigFileName is effectively the same as ConfigName, but includes our preferred extension
const ConfigFileName = ConfigName + ".yaml"

// ConfigFileType is our preferred file type for our config file, this must match ConfigFileName
const ConfigFileType = "yaml"

// EnvironmentEnvVarName is the name of the environment variable that specifies the current environment (dev, qa, prod, etc.)
const EnvironmentEnvVarName = "ACTIVESTATE_ENVIRONMENT"

// ProjectEnvVarName is the name of the environment variable that specifies the path of the activestate.yaml config file.
const ProjectEnvVarName = "ACTIVESTATE_PROJECT"

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

// APIHost is the host that we're talking against for our API
const APIHost = "platform.activestate.com"

// APIHostStaging is the host that we're talking against for our API when not on production
const APIHostStaging = "staging.activestate.com"

// APIHostTesting is the host that we're talking against for our API when not on production
const APIHostTesting = "testing.tld"

// APISchema is the schema used by our API
const APISchema = "https"

// APIPath is the path for our API endpoint
const APIPath = "/api/v1"

// APITokenName is the name we give our api token
const APITokenName = "activestate-platform-cli"

// ProductionBranch is the branch used for production builds
const ProductionBranch = "prod"
