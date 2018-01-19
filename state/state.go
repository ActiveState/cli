package main

import (
	"os"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/state/install"
	"github.com/jessevdk/go-flags"
)

var options struct {
	Locale func(string) `long:"locale" short:"L" description:"Locale"`
}

var parser = flags.NewNamedParser("state", flags.Default)

func init() {
	options.Locale = onSetLocale
}

func onSetLocale(localeName string) {
	locale.Set(localeName)
}

func main() {

	parser.AddGroup("Application Options", "", &options)

	_, err := parser.Parse()

	if err != nil {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func init() {
	command, shortDescription, longDescription, data := installCmd.Register()
	parser.AddCommand(command, shortDescription, longDescription, data)
}
