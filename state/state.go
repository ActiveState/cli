package main

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/state/install"
	"github.com/jessevdk/go-flags"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"
)

var options struct {
	Locale func(string) `long:"locale" short:"L" description:"Locale"`
}

var parser = flags.NewNamedParser("state", flags.Default)

func init() {
	configDirs := configdir.New("activestate", "cli")
	configDirs.LocalPath, _ = filepath.Abs(".")
	configDir := configDirs.QueryFolders(configdir.Global)[0]

	if !configDir.Exists("activestate.yaml") {
		configDir.Create("activestate.yaml")
	}

	viper.SetConfigName("activestate")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir.Path)
	viper.AddConfigPath(".")

	options.Locale = flagSetLocale

	parser.AddGroup("Application Options", "", &options)

	command, shortDescription, longDescription, data := installCmd.Register()
	parser.AddCommand(command, shortDescription, longDescription, data)
}

func main() {
	err := viper.ReadInConfig()
	if err != nil {
		print.Error("Fatal error while reading config file: %s \n", err)
		os.Exit(1)
	}

	_, err = parser.Parse()

	if err != nil {
		os.Exit(1)
	}

	viper.WriteConfig()
}

func flagSetLocale(localeName string) {
	locale.Set(localeName)
}
