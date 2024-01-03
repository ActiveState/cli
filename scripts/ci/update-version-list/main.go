package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/updater"
)

// Where the master version file lives on S3.
const S3PrefixURL = "https://state-tool.s3.amazonaws.com/"
const S3Bucket = "update/state/"
const VersionsJson = "versions.json"

// Valid channels to update the master version file with.
var ValidChannels = []string{constants.BetaChannel, constants.ReleaseChannel}

func init() {
	if !condition.OnCI() {
		// Allow testing with artifacts produced by `state run generate-test-update`
		ValidChannels = append(ValidChannels, "test-channel")
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <build-dir>", os.Args[0])
	}

	// Fetch the current master list from S3.
	versions := []updater.AvailableUpdate{}
	fmt.Printf("Fetching master %s file from S3\n", VersionsJson)
	bytes, err := httputil.Get(S3PrefixURL + S3Bucket + VersionsJson)
	if err != nil {
		log.Fatalf("Failed to fetch file: %s", err.Error())
	}
	err = json.Unmarshal(bytes, &versions)
	if err != nil {
		log.Fatalf("Failed to decode JSON: %s", err.Error())
	}

	// Find info.json files to add to the master list and add them.
	updated := false
	buildDir := os.Args[1]
	fmt.Printf("Searching for info.json files in %s\n", buildDir)
	files, err := fileutils.ListDir(buildDir, false)
	if err != nil {
		log.Fatalf("Failed to search %s: %s", buildDir, err.Error())
	}
	for _, file := range files {
		if file.Name() != "info.json" {
			continue
		}
		channel := strings.Split(file.RelativePath(), string(filepath.Separator))[0]
		if !funk.Contains(ValidChannels, channel) {
			continue
		}
		fmt.Printf("Found %s\n", file.RelativePath())
		bytes, err := fileutils.ReadFile(file.Path())
		if err != nil {
			log.Fatalf("Unable to read file: %s", err.Error())
		}
		info := updater.AvailableUpdate{}
		err = json.Unmarshal(bytes, &info)
		if err != nil {
			log.Fatalf("Unable to decode JSON: %s", err.Error())
		}
		info.Path = S3PrefixURL + S3Bucket + info.Path // convert relative path to full URL
		versions = append(versions, info)
		updated = true
	}

	if !updated {
		fmt.Println("No updates found.")
		return
	}

	// Write the updated list to disk. The s3-deployer script should pick it up and upload it.
	localVersionsJson := filepath.Join(buildDir, VersionsJson)
	fmt.Printf("Writing updated %s locally to %s\n", VersionsJson, localVersionsJson)
	bytes, err = json.Marshal(versions)
	if err != nil {
		log.Fatalf("Failed to encode JSON: %s", err.Error())
	}
	err = fileutils.WriteFile(localVersionsJson, bytes)
	if err != nil {
		log.Fatalf("Failed to write file: %s", err.Error())
	}
}
