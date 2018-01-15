package cmd

import (
	"os"

	"github.com/ActiveState/Zeridian-CLI/cmd/install"
	"github.com/jessevdk/go-flags"
)

var parser = flags.NewNamedParser("state", flags.Default|flags.HelpFlag)

// Execute the main command
func Execute() {
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
