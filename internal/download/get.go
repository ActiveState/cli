package download

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/sysinfo"
)

// Get takes a URL and returns the contents as bytes
var Get func(url string) ([]byte, *failures.Failure)

func init() {
	if flag.Lookup("test.v") != nil {
		Get = _testHTTPGet
	} else {
		Get = httpGet
	}
}

func httpGet(url string) ([]byte, *failures.Failure) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, failures.FailNetwork.Wrap(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logging.Debug("3")
		return nil, failures.FailNetwork.New("error_status_code", strconv.Itoa(resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logging.Debug("4")
		return nil, failures.FailIO.Wrap(err)
	}

	return body, nil
}

// _testHTTPGet is used when in tests, this cannot be in the test itself as that would limit it to only that one test
func _testHTTPGet(url string) ([]byte, *failures.Failure) {
	var OS = strings.ToLower(sysinfo.OS().String())
	var arch = strings.ToLower(sysinfo.Architecture().String())
	var platform = fmt.Sprintf("%s-%s", OS, arch)

	path := strings.Replace(url, constants.APIArtifactURL, "", 1)
	path = strings.Replace(path, "distro/"+platform+"/", "distro/", 1)
	path = filepath.Join(environment.GetRootPathUnsafe(), "test", path)

	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	return body, nil
}
