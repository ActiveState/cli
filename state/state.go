package main

import (
	"os"

	"github.com/ActiveState/ActiveState-CLI/state/install"
	"github.com/jessevdk/go-flags"
)

var parser = flags.NewNamedParser("state", flags.Default|flags.HelpFlag)

func main() {
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
