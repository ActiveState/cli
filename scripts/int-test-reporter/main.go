package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <source-dir>", os.Args[0])
	}
	sourceDir := os.Args[1]

	err := run(sourceDir)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
}

type operatingSystem int

type testFailures map[string]map[operatingSystem]string

type testOutput struct {
	Action string `json:"Action"`
	Test   string `json:"Test,omitempty"`
	Output string `json:"Output,omitempty"`
}

const (
	unknown operatingSystem = iota
	linux
	macOS
	windows
)

func getOutput(actions []testOutput, name string) string {
	var lines []string

	for _, a := range actions {
		if a.Test != name || a.Output == "" {
			continue
		}
		lines = append(lines, a.Output)
	}
	return strings.Join(lines, "")
}

func parseOS(osString string) (operatingSystem, error) {
	if osString == "Linux" {
		return linux, nil
	}
	if osString == "Windows" {
		return windows, nil
	}
	if osString == "macOS" {
		return macOS, nil
	}
	return unknown, fmt.Errorf("Invalid OS '%s'", osString)
}

func (o operatingSystem) String() string {
	if o == unknown {
		return "unknown"
	}
	if o == linux {
		return "linux"
	}
	if o == windows {
		return "windows"
	}
	if o == macOS {
		return "macOS"
	}
	return ""
}

func appendFailuresFromFile(sourceDir, f string, failures testFailures) (testFailures, error) {

	if !strings.HasSuffix(f, "-results.json") {
		return failures, nil
	}

	osString := strings.TrimSuffix(f, "-results.json")

	osys, err := parseOS(osString)
	if err != nil {
		log.Printf("error parsing os: %v", err)
		return failures, nil
	}

	// find failing tests
	buf, err := ioutil.ReadFile(filepath.Join(sourceDir, f))
	if err != nil {
		return failures, nil
	}

	lines := strings.Split(string(buf), "\n")
	actions := make([]testOutput, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			continue
		}
		var t testOutput
		err := json.Unmarshal([]byte(l), &t)
		if err != nil {
			return failures, fmt.Errorf("failed to parse %s: %v", l, err)
		}
		actions = append(actions, t)
	}

	for _, a := range actions {
		if a.Test == "" || a.Action != "fail" {
			continue
		}

		output := getOutput(actions, a.Test)
		if output == "" {
			continue
		}

		_, ok := failures[a.Test]
		if !ok {
			failures[a.Test] = make(map[operatingSystem]string)
		}
		failures[a.Test][osys] = output
	}

	return failures, nil
}

func alreadyReported(testName string) bool {
	return false
}

func report(testName string, outputs map[operatingSystem]string) {
	var osyses []string
	for osys := range outputs {
		osyses = append(osyses, osys.String())
	}

	fmt.Printf("test %s failed on oses: %+v with output:\n", testName, osyses)
}

func run(sourceDir string) error {
	failures := make(testFailures)

	files, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		failures, err = appendFailuresFromFile(sourceDir, f.Name(), failures)
		if err != nil {
			return err
		}
	}

	// filter summary tests
	var names []string
	isSummary := func(fn string) bool {
		for cfn := range failures {
			if strings.HasPrefix(strings.TrimPrefix(cfn, fn), "/") {
				return true
			}
		}
		return false
	}
	for fn := range failures {
		if isSummary(fn) {
			continue
		}
		names = append(names, fn)
	}

	// send failure reports
	for _, fn := range names {
		// check if already reported
		if alreadyReported(fn) {
			continue
		}

		report(fn, failures[fn])
	}

	return nil
}
