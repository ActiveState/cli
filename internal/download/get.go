package download

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/logging"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

// Get takes a URL and returns the contents as bytes
var Get func(url string) ([]byte, *failures.Failure)

// GetWithProgress takes a URL and returns the contents as bytes, it takes an optional second arg which will spawn a progressbar
var GetWithProgress func(url string, progress *mpb.Progress) ([]byte, *failures.Failure)

func init() {
	if flag.Lookup("test.v") != nil {
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

func httpGetWithProgress(url string, progress *mpb.Progress) ([]byte, *failures.Failure) {
	logging.Debug("Retrieving url: %s", url)
	resp, err := http.Head(url)
	if err != nil {
		return nil, failures.FailNetwork.Wrap(err)
	}
	length := resp.Header.Get("Content-Length")
	total, err := strconv.Atoi(length)
	if err != nil {
		logging.Debug("Content-length: %v", length)
		return nil, failures.FailInput.Wrap(err)
	}

	var bar *mpb.Bar
	if progress != nil {
		bar = progress.AddBar(int64(total),
			mpb.BarRemoveOnComplete(),
			mpb.PrependDecorators(
				decor.CountersKibiByte("%6.1f / %6.1f", 20, 0),
			),
			mpb.AppendDecorators(decor.Percentage(5, 0)))
	}

	resp, err = http.Get(url)
	if err != nil {
		return nil, failures.FailNetwork.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, failures.FailNetwork.New("error_status_code", strconv.Itoa(resp.StatusCode))
	}

	src := resp.Body
	var dst bytes.Buffer

	if progress != nil {
		src = bar.ProxyReader(resp.Body)
	}

	_, err = io.Copy(&dst, src)
	if err != nil {
		return nil, failures.FailInput.Wrap(err)
	}

	if progress != nil {
		if !bar.Completed() {
			// Failsafe, so we don't get blocked by a progressbar
			bar.IncrBy(total)
		}
	}

	return dst.Bytes(), nil
}

func _testHTTPGetWithProgress(url string, progress *mpb.Progress) ([]byte, *failures.Failure) {
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
