package svcctl

import (
	"context"
	"strings"

	"github.com/ActiveState/cli/internal/ipc"
)

var (
	KeyHTTPAddr  = "http-addr"
	KeyLogFile   = "log-file"
	KeyHeartBeat = "heart:"
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

func (c *Comm) GetHTTPAddr(ctx context.Context) (string, error) {
	return c.req.Request(ctx, KeyHTTPAddr)
}

func LogFileHandler(logFile string) ipc.RequestHandler {
	return func(input string) (string, bool) {
		if input == KeyLogFile {
			return logFile, true
		}
		return "", false
	}
}

func (c *Comm) GetLogFileName(ctx context.Context) (string, error) {
	return c.req.Request(ctx, KeyLogFile)
}

type RuntimeUsageReporter interface {
	ReportRuntimeUsage(ctx context.Context, pid, exec string)
}

func HeartBeatHandler(reporter RuntimeUsageReporter) ipc.RequestHandler {
	return func(input string) (string, bool) {
		if !strings.HasPrefix(input, KeyHeartBeat) {
			return "", false
		}

		data := input[len(KeyHeartBeat):]
		var pid, exec string

		ss := strings.Split(data, ":")
		if len(ss) > 0 {
			pid = ss[0]
		}
		if len(ss) > 1 {
			exec = ss[1]
		}

		reporter.ReportRuntimeUsage(context.Background(), pid, exec)

		return data, true
	}
}

func (c *Comm) SendHeartBeat(ctx context.Context, pid string) (string, error) {
	return c.req.Request(ctx, KeyHeartBeat+pid)
}
