package download

import (
	"context"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
)


func s3GetWithProgress(url *url.URL, progress *progress.Progress) ([]byte, error) {
	logging.Debug("Downloading via s3")

	s3m := parseS3URL(url)

	// Prepare AWS config
	config, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errs.Wrap(err, "Could not load default AWS config.")
	}
	config.Region = s3m.Region
	config.EndpointResolver = aws.ResolveWithEndpointURL(url.String())
	config.Credentials = aws.AnonymousCredentials

	// Read size of object
	s3sess := s3.New(config)
	headReq := s3sess.HeadObjectRequest(&s3.HeadObjectInput{
		Bucket: aws.String(s3m.Bucket),
		Key:    aws.String(s3m.Key),
	})
	headRes, err := headReq.Send(context.Background())
	if err != nil {
		return nil, locale.WrapError(err, "err_dl_s3head", "Requesting download meta information failed due to underlying S3 error: {{.V0}}.", err.Error())
	}
	var length int64
	if headRes != nil && headRes.ContentLength != nil {
		length = *headRes.ContentLength
	}

	// Record progress
	bar := progress.AddByteProgressBar(length)
	cb := func(length int) {
		if !bar.Completed() {
			// Failsafe, so we don't get blocked by a progressbar
			bar.IncrBy(length)
		}
	}

	// Prepare result
	b := []byte{}
	w := NewWriteAtBuffer(b, cb)

	// Download object
	dl := s3manager.NewDownloader(config)
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

func parseS3URL(url *url.URL) s3Meta {
	r := s3Meta{Key: url.Path}
	domain := strings.SplitN(url.Host, ".", 5)
	if strings.HasSuffix(url.Host, ".s3.amazonaws.com") { // https://bucket-name.s3.amazonaws.com/key-name
		r.Region = "us-east-1"
	} else if domain[1] == "s3" && domain[3] == "amazonaws" { // https://bucket-name.s3.Region.amazonaws.com/key-name
		r.Region = domain[2]
	} else { // https://s3.Region.amazonaws.com/bucket-name/key-name
		r.Bucket = strings.SplitN(url.Path, "/", 1)[0]
		r.Region = domain[1]
	}
	return r
}