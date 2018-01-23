package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"

	"github.com/ActiveState/ActiveState-CLI/state/install"

	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"
)

var rootCmd *cobra.Command

var Locale string

func init() {
	initConfig()
	initCommand()
}

func initConfig() {
	logging.Info("Init Config")

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

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}

	locale.Init()
}

func initCommand() {
	logging.Debug("Init Command")

	var T = locale.T
	var Tt = locale.Tt

	rootCmd = &cobra.Command{
		Use: "state",
		Run: Execute,
	}

	rootCmd.PersistentFlags().StringVarP(&Locale, "locale", "l", "", "localisation")

	// Small hack to parse the locale flag early, so we can localise our commands properly
	args := os.Args[1:]
	_, flags, err := rootCmd.Traverse(args)
	if err == nil {
		_ = rootCmd.ParseFlags(flags)

		if Locale != "" {
			locale.Set(Locale)
		}
	}

	rootCmd.SetShort(T("state_description"))
	rootCmd.SetUsageTemplate(Tt("usage_tpl"))

	install.Register(rootCmd)
}

func main() {
	logging.Debug("main")

	err := rootCmd.Execute()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if Locale != "" {
		viper.Set("Locale", Locale)
	}

	viper.WriteConfig()
}

func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	print.Line(viper.GetString("Locale"))
}
