package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

var (
	installerFile             string
	findString, replaceString *string
)

func init() {
	findString = flag.String("f", "", "The string to replace")
	replaceString = flag.String("r", "", "The string to replace with")
}

func main() {
	if !condition.InUnitTest() {
		err := run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s error: %v", os.Args[0], errs.Join(err, ":"))
		}
	}
}

func printUsage() {
	fmt.Println("")
	fmt.Println("[-f findString] [-r replaceString] <installerFile>")
}

func run() error {
	flag.Parse()
	if flag.NArg() < 1 && !condition.InUnitTest() {
		flag.Usage()
		printUsage()
		os.Exit(0)
	}

	installerFile = flag.Args()[0]
	err := fileutils.ReplaceAll(installerFile, *findString, *replaceString)
	if err != nil {
		return errs.Wrap(err, "Could not replace text in file: %s", installerFile)
	}

	return nil
}
