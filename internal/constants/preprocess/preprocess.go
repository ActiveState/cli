package preprocess

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
)

// Constants holds constants that will be preprocessed, meaning the key value parts here will be built into the constants
// package as actual constants, using the build-time interpretations
var Constants = map[string]func() string{}

type nestedStringMap map[string]map[string]string

var envSettings = nestedStringMap{
	"prod": {
		"APISchema":        "https",
		"APIHost":          "platform.activestate.com",
		"SecretsAPISchema": "https",
		"SecretsAPIHost":   "platform.activestate.com",
	},
	"stage": {
		"APISchema":        "https",
		"APIHost":          "staging.activestate.build",
		"SecretsAPISchema": "https",
		"SecretsAPIHost":   "staging.activestate.build",
	},
	"dev": {
		"APISchema":        "https",
		"APIHost":          "staging.activestate.build",
		"SecretsAPISchema": "http",
		"SecretsAPIHost":   "localhost:8080",
	},
	"test": {
		"APISchema":        "https",
		"APIHost":          "testing.tld",
		"SecretsAPISchema": "https",
		"SecretsAPIHost":   "testing.tld",
	},
}

func init() {
	Constants["BranchName"] = getBranchName
	Constants["BuildNumber"] = func() string {
		out := getCmdOutput("git rev-list --abbrev-commit HEAD")
		return strconv.Itoa(len(strings.Split(out, "\n")))
	}
	Constants["RevisionHash"] = func() string { return getCmdOutput("git rev-parse --verify HEAD") }
	Constants["Version"] = func() string { return fmt.Sprintf("%s-%s", constants.VersionNumber, Constants["BuildNumber"]()) }
	Constants["Date"] = func() string { return time.Now().Format("Mon Jan 2 2006 15:04:05 -0700 MST") }

	Constants["Env"] = func() string { return getEnvName() }
	Constants["APISchema"] = func() string { return envSettings[getEnvName()]["APISchema"] }
	Constants["APIHost"] = func() string { return envSettings[getEnvName()]["APIHost"] }
	Constants["SecretsAPISchema"] = func() string { return envSettings[getEnvName()]["SecretsAPISchema"] }
	Constants["SecretsAPIHost"] = func() string { return envSettings[getEnvName()]["SecretsAPIHost"] }
}

func getCmdOutput(cmdString string) string {
	cmdArgs := strings.Split(cmdString, " ")

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		log.Fatalf("Command failed, command: %s, args: %v, output: %s, error: %s, code: %s", cmdArgs[0], cmdArgs[1:], out.String(), stderr.String(), err)
	}
	return strings.Trim(out.String(), "\n")
}

var branchName string

func getBranchName() string {
	if branchName == "" {
		branchName = getCmdOutput("git rev-parse --abbrev-ref HEAD")
	}
	return branchName
}

var envName string

func getEnvName() string {
	if envName == "" {
		envName = strings.TrimSpace(os.Getenv("CLIENV"))
		if _, hasSetting := envSettings[envName]; hasSetting {
			return envName
		}

		if flag.Lookup("test.v") != nil {
			envName = "test"
		} else if bn := getBranchName(); bn == "prod" {
			envName = "prod"
		} else {
			envName = "stage"
		}
	}
	return envName
}
