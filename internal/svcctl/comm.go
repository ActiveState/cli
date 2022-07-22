package svcctl

import (
	"context"
	"strings"

	"github.com/ActiveState/cli/internal/ipc"
)

var (
	KeyHTTPAddr  = "http-addr"
	KeyLogFile   = "log-file"
	KeyHeartbeat = "heart<"
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
	ReportRuntimeUsage(ctx context.Context, pid, exec, namespace, commit, headless string)
}

func HeartbeatHandler(reporter RuntimeUsageReporter) ipc.RequestHandler {
	return func(input string) (string, bool) {
		// format : heart<{proc-id}<{exec-path}<{namespace}<{commit-id}<{headless-bool}
		// example: heart<123</home/user/.local/dir/beta/bin/state-exec<org/prj<1234abcd-1234-abcd-1234-abcd1234abcd<false
		if !strings.HasPrefix(input, KeyHeartbeat) {
			return "", false
		}

		data := input[len(KeyHeartbeat):]
		var pid, exec, namespace, commit, headless string

		ss := strings.SplitN(data, "<", 5)
		if len(ss) > 0 {
			pid = ss[0]
		}
		if len(ss) > 1 {
			exec = ss[1]
		}
		if len(ss) > 2 {
			namespace = ss[2]
		}
		if len(ss) > 3 {
			commit = ss[3]
		}
		if len(ss) > 4 {
			headless = ss[4]
		}

		reporter.ReportRuntimeUsage(context.Background(), pid, exec, namespace, commit, headless)

		return data, true
	}
}

func (c *Comm) SendHeartbeat(ctx context.Context, pid string) (string, error) {
	return c.req.Request(ctx, KeyHeartbeat+pid)
}
