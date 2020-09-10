package download

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
)

func s3GetWithProgress(url *url.URL, progress *progress.Progress) ([]byte, error) {
	logging.Debug("Downloading via s3")

	s3m := parseS3URL(url)

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
	} else if res.StatusCode != http.StatusOK {
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

	bar.Abort(true) // ensure we don't get stuck on an incomplete bar

	return w.Bytes(), nil
}

type s3Meta struct {
	Bucket string
	Region string
	Key    string
}

func parseS3URL(url *url.URL) s3Meta {
	r := s3Meta{Key: url.Path}
	domain := strings.SplitN(url.Host, ".", 5)
	if strings.HasSuffix(url.Host, ".s3.amazonaws.com") { // https://bucket-name.s3.amazonaws.com/key-name
		r.Bucket = domain[0]
		r.Region = "us-east-1"
	} else if domain[1] == "s3" && domain[3] == "amazonaws" { // https://bucket-name.s3.Region.amazonaws.com/key-name
		r.Bucket = domain[0]
		r.Region = domain[2]
	} else { // https://s3.Region.amazonaws.com/bucket-name/key-name
		r.Bucket = strings.SplitN(url.Path, "/", 1)[0]
		r.Region = domain[1]
	}
	return r
}