package legacyupd

import (
	"context"
	"io"
	"net/http"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/pkg/errors"
)

// Fetch will return an HTTP request to the specified url and return
// the body of the result. An error will occur for a non 200 status code.
func Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errs.Wrap(err, "Could not init get request for %s", url)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errs.Wrap(err, "Couldn't get url=%s", url)
	}

	if resp.StatusCode != 200 {
		return nil, errors.Errorf(
			"bad http status from %s: %v",
			url, resp.Status)
	}

	return resp.Body, nil
}
