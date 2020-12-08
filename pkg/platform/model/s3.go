package model

import (
	"net/url"

	"github.com/go-openapi/strfmt"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/s3"
)

var (
	FailSignS3URL = failures.Type("model.fail.signs3url", api.FailUnknown)

	FailInvalidURL = failures.Type("model.fail.invalidurl")
)

func SignS3URL(u *url.URL) (*url.URL, error) {
	unescaped, err := url.QueryUnescape(u.String())
	if err != nil {
		return nil, FailInvalidURL.Wrap(err)
	}

	params := s3.NewSignS3URIParams()
	params.URI = strfmt.URI(unescaped)

	res, err := mono.Get().S3.SignS3URI(params)
	if err != nil {
		return nil, FailSignS3URL.Wrap(err)
	}

	ur, err := url.Parse(res.Payload.URI.String())
	if err != nil {
		logging.Error("API responded with an invalid url: %s, error: %v", res.Payload.URI.String(), err)
		return ur, FailInvalidURL.New(locale.T("err_signs3_invalid_url"))
	}

	return ur, nil
}
