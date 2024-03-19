package workflow_helpers

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/google/go-github/v45/github"
)

/*
Code based on: https://github.com/hashicorp/terraform-provider-github/blob/fa73654b66e37b1fd8d886141d9c2974e24ba42f/github/transport.go#L42-L109
Sourced via: https://github.com/google/go-github/issues/431

MIT License

Copyright (c) 2020 GitHub

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

const (
	ctxId      = "id"
	writeDelay = 720 * time.Millisecond // https://github.com/google/go-github/issues/431#issuecomment-248767702
	// The above writeDelay is the minimum interval to avoid rate limiting as long as is only one
	// job using the GitHub API at a time. Since this is likely not the case for us, we need a tunable
	// parameter that can easily be adjusted based on experience.
	multiplier = 1.5 // 150% of the minimum delay
)

// rateLimitTransport implements GitHub's best practices
// for avoiding rate limits
// https://developer.github.com/v3/guides/best-practices-for-integrators/#dealing-with-abuse-rate-limits
type rateLimitTransport struct {
	transport        http.RoundTripper
	delayNextRequest bool
	accessToken      string

	m sync.Mutex
}

func (rlt *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+rlt.accessToken)
	// Make requests for a single user or client ID serially
	// This is also necessary for safely saving
	// and restoring bodies between retries below
	rlt.lock(req)

	// If you're making a large number of POST, PATCH, PUT, or DELETE requests
	// for a single user or client ID, wait at least one second between each request.
	if rlt.delayNextRequest {
		delay := time.Duration(multiplier * float64(writeDelay))
		logging.Debug("Sleeping %s between write operations", delay)
		time.Sleep(delay)
	}

	rlt.delayNextRequest = isWriteMethod(req.Method)

	resp, err := rlt.transport.RoundTrip(req)
	if err != nil {
		rlt.unlock(req)
		return resp, err
	}

	// Make response body accessible for retries & debugging
	// (work around bug in GitHub SDK)
	// See https://github.com/google/go-github/pull/986
	r1, r2, err := drainBody(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = r1
	ghErr := github.CheckResponse(resp)
	resp.Body = r2

	// When you have been limited, use the Retry-After response header to slow down.
	if arlErr, ok := ghErr.(*github.AbuseRateLimitError); ok {
		rlt.delayNextRequest = false
		retryAfter := arlErr.GetRetryAfter()
		logging.Debug("Abuse detection mechanism triggered, sleeping for %s before retrying",
			retryAfter)
		time.Sleep(retryAfter)
		rlt.unlock(req)
		return rlt.RoundTrip(req)
	}

	if rlErr, ok := ghErr.(*github.RateLimitError); ok {
		rlt.delayNextRequest = false
		retryAfter := rlErr.Rate.Reset.Sub(time.Now())
		logging.Debug("Rate limit %d reached, sleeping for %s (until %s) before retrying",
			rlErr.Rate.Limit, retryAfter, time.Now().Add(retryAfter))
		time.Sleep(retryAfter)
		rlt.unlock(req)
		return rlt.RoundTrip(req)
	}

	rlt.unlock(req)

	return resp, nil
}

func (rlt *rateLimitTransport) lock(req *http.Request) {
	ctx := req.Context()
	logging.Debug("[TRACE] Aquiring lock for GitHub API request (%q)", ctx.Value(ctxId))
	rlt.m.Lock()
}

func (rlt *rateLimitTransport) unlock(req *http.Request) {
	ctx := req.Context()
	logging.Debug("[TRACE] Releasing lock for GitHub API request (%q)", ctx.Value(ctxId))
	rlt.m.Unlock()
}

func NewRateLimitTransport(rt http.RoundTripper, accessToken string) *rateLimitTransport {
	return &rateLimitTransport{transport: rt, accessToken: accessToken}
}

// drainBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func isWriteMethod(method string) bool {
	switch method {
	case "POST", "PATCH", "PUT", "DELETE":
		return true
	}
	return false
}
