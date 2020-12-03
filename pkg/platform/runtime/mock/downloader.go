package mock

import (
	"github.com/ActiveState/cli/internal/failures"
	testifyMock "github.com/stretchr/testify/mock"
)

// Downloader is a testify Mock object.
type Downloader struct {
	testifyMock.Mock
}

// NewMockDownloader returns a new testify/mock.Mock Downloader.
func NewMockDownloader() *Downloader {
	return &Downloader{}
}

// Download for Downloader.
func (downloader *Downloader) Download() (string, error) {
	args := downloader.Called()
	if failure := args.Get(1); failure != nil {
		return args.String(0), failure.(error)
	}
	return args.String(0), nil
}
