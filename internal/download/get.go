package download

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
)

// GetWithProgress takes a URL and returns the contents as bytes, it takes an optional second arg which will spawn a progressbar
var GetWithProgress func(url *url.URL, progress *progress.Progress) ([]byte, error)

func init() {
	SetMocking(condition.InTest())
}

// SetMocking sets the correct Get methods for testing
func SetMocking(useMocking bool) {
	if useMocking {
		GetWithProgress = _testGetWithProgress
	} else {
		GetWithProgress = s3GetWithProgress
	}
}

func s3GetWithProgress(url *url.URL, progress *progress.Progress) ([]byte, error) {
	logging.Debug("Downloading via s3")

	s3m, err := parseS3URL(url)
	if err != nil {
		return nil, locale.WrapError(err, "err_s3_parseurl", "Could not parse the artifact URL.")
	}

	// Prepare AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s3m.Region),
		Credentials: credentials.AnonymousCredentials,
	})
	if err != nil {
		return nil, errs.Wrap(err, "Could not create aws session")
	}

	// Grab file size
	var length int64 = 0
	res, err := http.Get(url.String())
	if err != nil {
		logging.Debug("Could not grab url: %v", err)
	} else if res != nil && res.StatusCode != http.StatusOK {
		logging.Debug("Could not grab url due to statuscode: %d", res.StatusCode)
	} else {
		lengthInt, err := strconv.Atoi(res.Header.Get("Content-Length"))
		if err != nil {
			logging.Debug("Could not grab content-length: %v", err)
		} else {
			length = int64(lengthInt)
		}
	}

	// Close early cause we're just looking at the header
	// Yes normally you'd use a HEAD for this, but S3 presigned URLs don't support HEAD requests
	res.Body.Close()

	// Record progress
	bar := progress.AddByteProgressBar(length)
	defer bar.Abort(true) // ensure we don't get stuck on an incomplete bar

	cb := func(length int) {
		if !bar.Completed() {
			// Failsafe, so we don't get blocked by a progressbar
			bar.IncrBy(length)
		}
	}

	// Prepare result recorder
	b := []byte{}
	w := NewWriteAtBuffer(b, cb)

	dl := s3manager.NewDownloader(sess)
	dl.RequestOptions = append(dl.RequestOptions, func(r *request.Request) {
		r.Handlers.Build.PushBack(func(r *request.Request) {
			// Work around AWS rewriting our query in the wrong order, causing signing to fail
			r.HTTPRequest.URL.RawQuery = url.RawQuery
		})
	})
	_, err = dl.Download(w,
		&s3.GetObjectInput{
			Bucket: aws.String(s3m.Bucket),
			Key:    aws.String(s3m.Key),
		})
	if err != nil {
		return nil, locale.WrapError(err, "err_dl_s3", "Downloading failed due to underlying S3 error: {{.V0}}.", err.Error())
	}

	return w.Bytes(), nil
}

type s3Meta struct {
	Bucket string
	Region string
	Key    string
}

func parseS3URL(url *url.URL) (s3Meta, error) {
	r := s3Meta{Key: url.Path}
	domain := strings.SplitN(url.Host, ".", 4)
	if len(domain) != 4 {
		return r, locale.NewError("err_s3_host", "API responded with an invalid artifact host: {{.V0}}.", url.Host)
	}

	// https://bucket-name.s3.amazonaws.com/key-name
	if strings.HasSuffix(url.Host, ".s3.amazonaws.com") {
		return s3Meta{
			domain[0],
			constants.DefaultS3Region,
			url.Path,
		}, nil
	}

	// https://bucket-name.s3.Region.amazonaws.com/key-name
	if domain[1] == "s3" && domain[3] == "amazonaws.com" {
		return s3Meta{
			domain[0],
			domain[2],
			url.Path,
		}, nil
	}

	// https://s3.Region.amazonaws.com/bucket-name/key-name
	path := strings.Split(url.Path, "/")
	return s3Meta{
		path[1],
		domain[1],
		fmt.Sprintf("/%s", strings.Join(path[2:len(path)], "/")),
	}, nil
}

func _testGetWithProgress(url *url.URL, progress *progress.Progress) ([]byte, error) {
	return _testGet(url)
}

// _testGet is used when in tests, this cannot be in the test itself as that would limit it to only that one test
func _testGet(url *url.URL) ([]byte, error) {
	path := strings.Replace(url.String(), constants.APIArtifactURL, "", 1)
	path = filepath.Join(environment.GetRootPathUnsafe(), "test", path)

	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return body, nil
}