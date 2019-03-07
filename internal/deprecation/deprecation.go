package deprecation

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-version"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
)

// DefaultTimeout defines how long we should wait for a response from constants.DeprecationInfoURL
const DefaultTimeout = time.Second

var (
	// FailFetchDeprecationInfo communicates a failure in retrieving the deprecation info via http
	FailFetchDeprecationInfo = failures.Type("deprecation.fail.info", failures.FailNetwork)

	// FailParseVersion communicates a failure in parsing a semantic version (the version is not formatted properly)
	FailParseVersion = failures.Type("deprecation.fail.versionparse", failures.FailInput)

	// FailTimeout communicates a failure due to a timeout
	FailTimeout = failures.Type("deprecation.fail.timeout", failures.FailNetwork)
)

// Info details deprecation information for a given version
type Info struct {
	Version     string `json:"version"`
	versionInfo *version.Version
	Date        time.Time `json:"date"`
	DateReached bool
	Reason      string `json:"reason"`
}

// Checker is the struct that we use to do checks with
type Checker struct {
	timeout time.Duration
}

// NewChecker returns a new instance of the Checker struct
func NewChecker(timeout time.Duration) *Checker {
	return &Checker{timeout}
}

// Check will run a Checker.Check with defaults
func Check() (*Info, *failures.Failure) {
	checker := NewChecker(DefaultTimeout)
	return checker.Check()
}

// Check will check if the current version of the tool is deprecated and returns deprecation info if it is.
// This uses a fairly short timeout to check against our deprecation url, so this should not be considered conclusive.
func (checker *Checker) Check() (*Info, *failures.Failure) {
	infos, fail := checker.fetchDeprecationInfo()
	if fail != nil {
		return nil, fail
	}

	versionInfo, err := version.NewVersion(constants.VersionNumber)
	if err != nil {
		return nil, FailParseVersion.Wrap(err)
	}

	for _, info := range infos {
		if versionInfo.LessThan(info.versionInfo) || versionInfo.Equal(info.versionInfo) {
			return &info, nil
		}
	}

	return nil, nil
}

func (checker *Checker) fetchDeprecationInfoBody() ([]byte, *failures.Failure) {
	client := http.Client{
		Timeout: time.Duration(checker.timeout),
	}

	resp, err := client.Get(constants.DeprecationInfoURL)
	if err != nil {
		// Check for timeout by evaluating the error string. Yeah this is dumb, thank the http package for that.
		if strings.Contains(err.Error(), "Client.Timeout") {
			return nil, FailTimeout.Wrap(err)
		}
		return nil, FailFetchDeprecationInfo.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	return body, nil
}

func (checker *Checker) fetchDeprecationInfo() ([]Info, *failures.Failure) {
	body, fail := checker.fetchDeprecationInfoBody()
	if fail != nil {
		return nil, fail
	}

	infos := make([]Info, 0)
	err := json.Unmarshal(body, &infos)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	for k := range infos {
		infos[k].versionInfo, err = version.NewVersion(infos[k].Version)
		if err != nil {
			return nil, FailParseVersion.Wrap(err)
		}
		infos[k].DateReached = infos[k].Date.Before(time.Now())
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].versionInfo.GreaterThan(infos[j].versionInfo)
	})

	return infos, nil
}
