package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/pkg/errors"
)

func printUsage() {
	fmt.Println("")
	fmt.Println("<outputPath> The path to output version.json to")
}

func main() {
	if !condition.InTest() {
		run()
	}
}

type current struct {
	Version string
}

func run() {
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		os.Exit(0)
	}

	outputPath := flag.Arg(0)
	err := os.MkdirAll(outputPath, 0755)
	if err != nil {
		panic(errors.Wrap(err, "Could not create directory for version.json file"))
	}

	versionPath := filepath.Join(outputPath, "version.json")
	file, err := os.Create(versionPath)
	if err != nil {
		panic(errors.Wrap(err, "Could not create version file"))
	}

	c := current{Version: constants.Version}
	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		fmt.Println("error:", err)
	}

	fmt.Printf("Updating version file at %s\n", versionPath)
	err = ioutil.WriteFile(file.Name(), b, 0755)
	if err != nil {
		panic(err)
	}
}
