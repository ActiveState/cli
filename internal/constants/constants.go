package constants

// LibraryName contains the main name of this library
const LibraryName = "ActiveState-CLI"

// LibraryNamespace is the namespace that the library belongs to
const LibraryNamespace = "github.com/ActiveState/"

// ConfigNamespace holds the appdata folder name under which we store our config
const ConfigNamespace = "activestate"

// ConfigName is used to inform viper and our config lib about the name of the config file
const ConfigName = "activestate"

// ConfigFileName is effectively the same as ConfigName, but includes our preferred extension
const ConfigFileName = ConfigName + ".yaml"

// ConfigFileType is our preferred file type for our config file, this must match ConfigFileName
const ConfigFileType = "yaml"
