package preprocess

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	
)

// Constants holds constants that will be preprocessed, meaning the key value parts here will be built into the constants
// package as actual constants, using the build-time interpretations
var Constants = map[string]func() string{}

func init() {
	status := workspaceStatus()

	Constants["BranchName"] = func() string { return status["STABLE_BRANCHNAME"] }
	Constants["BuildNumber"] = func() string { return strings.Trim(status["STABLE_BUILDNUMBER"], " ") }
	Constants["RevisionHash"] = func() string { return status["STABLE_REVISIONHASH"] }
	Constants["Version"] = func() string { return fmt.Sprintf("%s-%s", constants.VersionNumber, Constants["BuildNumber"]()) }
}

func workspaceStatus() map[string]string {
	text, err := ioutil.ReadFile(filepath.Join("bazel-out", "stable-status.txt"))
	if err != nil {
		log.Fatal(err)
	}

	result := map[string]string{}
	lines := strings.Split(string(text), "\n")
	for _, line := range lines {
		keyval := strings.Split(line, " ")
		result[keyval[0]] = strings.Join(keyval[1:], " ")
	}

	return result
}
