package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/pkg/errors"
)

func printUsage() {
	fmt.Println("")
	fmt.Println("<outputPath> The path to output master-version.json to")
}

func main() {
	if !condition.InTest() {
		run()
	}
}

func run() {
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		os.Exit(0)
	}

	resp, err := http.Get("https://s3.ca-central-1.amazonaws.com/cli-update/update/state/version.json")
	if err != nil {
		panic(errors.Wrap(err, "Could not get version file from S3"))
	}
	defer resp.Body.Close()

	outputPath := flag.Arg(0)
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		panic(errors.Wrap(err, "Could not create directory for version.json file"))
	}

	file, err := os.Create(filepath.Join(outputPath, "base-version.json"))
	if err != nil {
		panic(errors.Wrap(err, "Could not create version file"))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(errors.Wrap(err, "Could not read response body"))
	}

	err = ioutil.WriteFile(file.Name(), body, 0644)
	if err != nil {
		panic(errors.Wrap(err, "Could not write version data to file"))
	}
}
