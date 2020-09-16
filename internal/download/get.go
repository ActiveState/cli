package download

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/retryhttp"
)

// Get takes a URL and returns the contents as bytes
var Get func(url string) ([]byte, *failures.Failure)

// GetWithProgress takes a URL and returns the contents as bytes, it takes an optional second arg which will spawn a progressbar
var GetWithProgress func(url string, progress *progress.Progress) ([]byte, *failures.Failure)

func init() {
	SetMocking(condition.InTest())
}

// SetMocking sets the correct Get methods for testing
func SetMocking(useMocking bool) {
	if useMocking {
		Get = _testHTTPGet
		GetWithProgress = _testHTTPGetWithProgress
	} else {
		Get = httpGet
		GetWithProgress = httpGetWithProgress
	}
}

func httpGet(url string) ([]byte, *failures.Failure) {
	logging.Debug("Retrieving url: %s", url)
	return httpGetWithProgress(url, nil)
}

func httpGetWithProgress(url string, progress *progress.Progress) ([]byte, *failures.Failure) {
	return httpGetWithProgressRetry(url, progress, 3)
}

func httpGetWithProgressRetry(url string, progress *progress.Progress, attempt int) ([]byte, *failures.Failure) {
	logging.Debug("Retrieving url: %s, attempt: %d", url, attempt)
	client := retryhttp.NewClient(0 /* 0 = no timeout */, 3)
	resp, err := client.Get(url)
	if err != nil {
		code := -1
		if resp != nil {
			code = resp.StatusCode
		}
		return nil, failures.FailNetwork.Wrap(err, locale.Tl("err_network_get", "Status code: {{.V0}}", strconv.Itoa(code)))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, failures.FailNetwork.New("err_invalid_status_code", strconv.Itoa(resp.StatusCode))
	}

	var total int
	length := resp.Header.Get("Content-Length")
	if length == "" {
		total = 1
	} else {
		total, err = strconv.Atoi(length)
		if err != nil {
			logging.Debug("Content-length: %v", length)
			return nil, failures.FailInput.Wrap(err)
		}
	}

	bar := progress.AddByteProgressBar(int64(total))
	
	// Ensure bar is always closed (especially for retries)
	defer func() {
		if !bar.Completed() {
			bar.Abort(true)
		}
	}()

	src := resp.Body
	var dst bytes.Buffer

	src = bar.ProxyReader(resp.Body)

	_, err = io.Copy(&dst, src)
	if err != nil {
		logging.Debug("Reading body failed: %s", err)
		if attempt < 3 {
			return httpGetWithProgressRetry(url, progress, attempt + 1)
		}
		return nil, failures.FailNetwork.Wrap(err)
	}

	if !bar.Completed() {
		// Failsafe, so we don't get blocked by a progressbar
		bar.IncrBy(total)
	}

	return dst.Bytes(), nil
}

func _testHTTPGetWithProgress(url string, progress *progress.Progress) ([]byte, *failures.Failure) {
	return _testHTTPGet(url)
}

// _testHTTPGet is used when in tests, this cannot be in the test itself as that would limit it to only that one test
func _testHTTPGet(url string) ([]byte, *failures.Failure) {
	path := strings.Replace(url, constants.APIArtifactURL, "", 1)
	path = filepath.Join(environment.GetRootPathUnsafe(), "test", path)

	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	return body, nil
}
