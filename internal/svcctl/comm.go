package svcctl

import (
	"context"

	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/logging"
)

var (
	KeyHTTPAddr = "http-addr"
	KeyLogFile  = "log-file"
)

type Requester interface {
	Request(ctx context.Context, key string) (value string, err error)
}

type Comm struct {
	req Requester
}

func NewComm(req Requester) *Comm {
	return &Comm{
		req: req,
	}
}

func HTTPAddrHandler(addr string) ipc.RequestHandler {
	return func(input string) (string, bool) {
		if input == KeyHTTPAddr {
			return addr, true
		}

		return "", false
	}
}

func LogFileHandler(logFile string) ipc.RequestHandler {
	return func(input string) (string, bool) {
		if input == KeyLogFile {
			return logging.FileName(), true
		}
		return "", false
	}
}

func (c *Comm) GetHTTPAddr(ctx context.Context) (string, error) {
	return c.req.Request(ctx, KeyHTTPAddr)
}

func (c *Comm) GetLogFile(ctx context.Context) (string, error) {
	return c.req.Request(ctx, KeyLogFile)
}
