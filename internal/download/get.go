package download

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/retryhttp"
)

func init() {
}

func httpGetWithProgress(url string, progress *progress.Progress) ([]byte, *failures.Failure) {
	logging.Debug("Downloading via https")
	return nil, failures.FailNetwork.New("Wrong getter")
	logging.Debug("Retrieving url: %s", url)
	client := retryhttp.NewClient(0 /* 0 = no timeout */, 5)
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

	src := resp.Body
	var dst bytes.Buffer

	src = bar.ProxyReader(resp.Body)

	_, err = io.Copy(&dst, src)
	if err != nil {
		return nil, failures.FailInput.Wrap(err)
	}

	if !bar.Completed() {
		// Failsafe, so we don't get blocked by a progressbar
		bar.IncrBy(total)
	}

	return dst.Bytes(), nil
}

// RoundTripper is an implementation of http.RoundTripper that adds additional request information
type RoundTripper struct {
	params url.Values
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (r *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Amz-Algorithm", r.params.Get("X-Amz-Algorithm"))
	req.Header.Set("X-Amz-Credential", r.params.Get("X-Amz-Credential"))
	req.Header.Set("X-Amz-Signature", r.params.Get("X-Amz-Signature"))
	return http.DefaultTransport.RoundTrip(req)
}

func s3GetWithProgress(url *url.URL, progress *progress.Progress) ([]byte, error) {
	logging.Debug("Downloading via s3")

	query := url.Query()

	sess, err := session.NewSession(&aws.Config{
		Region:                        aws.String("us-east-1"),
		CredentialsChainVerboseErrors: aws.Bool(true),
		HTTPClient:                    &http.Client{Transport: &RoundTripper{query}},
		Credentials:                   credentials.NewStaticCredentials(query.Get("X-Amz-Credential"), query.Get("X-Amz-Signature"), ""),
	})
	if err != nil {
		return nil, errs.Wrap(err, "Could not create aws session")
	}

	domain := strings.SplitN(url.Host, ".", 2)
	bucket := domain[0]
	key := url.Path

	s3sess := s3.New(sess)
	head, err := s3sess.HeadObject(&s3.HeadObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String(key),
		SSECustomerAlgorithm: aws.String(query.Get("X-Amz-Algorithm")),
		SSECustomerKey:       aws.String(query.Get("X-Amz-Credential")),
		SSECustomerKeyMD5:    aws.String(query.Get("X-Amz-Signature")),
	})
	if err != nil {
		return nil, locale.WrapError(err, "err_dl_s3head", "Requesting download meta information failed due to underlying S3 error: {{.V0}}.", err.Error())
	}

	var length int64
	if head != nil && head.ContentLength != nil {
		length = *head.ContentLength
	}

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

	downloader := s3manager.NewDownloader(sess)
	_, err = downloader.Download(w,
		&s3.GetObjectInput{
			Bucket:               aws.String(bucket),
			Key:                  aws.String(key),
			SSECustomerAlgorithm: aws.String(query.Get("X-Amz-Algorithm")),
			SSECustomerKey:       aws.String(query.Get("X-Amz-Credential")),
			SSECustomerKeyMD5:    aws.String(query.Get("X-Amz-Signature")),
		})
	if err != nil {
		return nil, locale.WrapError(err, "err_dl_s3", "Downloading failed due to underlying S3 error: {{.V0}}.", err.Error())
	}

	return w.Bytes(), nil
}
