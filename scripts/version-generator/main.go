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
	fmt.Println("<branch> <outputPath> The path to output version.json to")
}

func main() {
	if !condition.InTest() {
		run()
	}
}

func run() {
	flag.Parse()
	if flag.NArg() != 2 {
		printUsage()
		os.Exit(0)
	}
	branch := flag.Arg(0)

	stateURL := "https://s3.ca-central-1.amazonaws.com/cli-update/update/state/%s/linux-amd64.json"
	resp, err := http.Get(fmt.Sprintf(stateURL, branch))
	if err != nil {
		panic(errors.Wrap(err, "Could not get version file from S3"))
	}
	defer resp.Body.Close()

	outputPath := flag.Arg(1)
	file, err := os.Create(filepath.Join(outputPath, "version.json"))
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
