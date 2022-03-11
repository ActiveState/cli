package model

import (
	"net/url"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/s3"
)

func SignS3URL(u *url.URL) (*url.URL, error) {
	params := s3.NewSignS3URIParams()
	params.URI = strfmt.URI(u.String())

	res, err := mono.Get().S3.SignS3URI(params)
	if err != nil {
		return nil, errs.Wrap(err, "SignS3URL failure")
	}

	ur, err := url.Parse(res.Payload.URI.String())
	if err != nil {
		logging.Error("API responded with an invalid url: %s, error: %v", res.Payload.URI.String(), err)
		rollbar.Error("API responded with an invalid url: %s, error: %v", res.Payload.URI.String(), err)
		return ur, locale.NewError("InvalidURL")
	}

	return ur, nil
}
