package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
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

func run() {
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		os.Exit(0)
	}
	branch := constants.BranchName
	if branchOverride, ok := os.LookupEnv("BRANCH_OVERRIDE"); ok {
		branch = branchOverride
	}

	stateURL := "https://s3.ca-central-1.amazonaws.com/cli-update/update/state/%s/linux-amd64.json"
	resp, err := http.Get(fmt.Sprintf(stateURL, branch))
	if err != nil {
		panic(errors.Wrap(err, "Could not get version file from S3"))
	}
	defer resp.Body.Close()

	outputPath := flag.Arg(0)
	fail := fileutils.Touch(filepath.Join(outputPath, "version.json"))
	if fail != nil {
		panic(errors.Wrap(fail.ToError(), "Could not create version file"))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(errors.Wrap(err, "Could not read response body"))
	}

	fail = fileutils.WriteFile(filepath.Join(outputPath, "version.json"), body)
	if fail != nil {
		panic(errors.Wrap(fail.ToError(), "Could not write version data to file"))
	}
}
