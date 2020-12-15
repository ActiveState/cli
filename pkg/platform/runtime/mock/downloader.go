package mock

import (
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
	if err := args.Get(1); err != nil {
		return args.String(0), err.(error)
	}
	return args.String(0), nil
}
