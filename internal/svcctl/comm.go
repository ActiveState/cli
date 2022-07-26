package svcctl

import (
	"context"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
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
	RuntimeUsage(ctx context.Context, pid int, exec, dimensionsJSON string) (*graph.RuntimeUsageResponse, error)
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

		pidNum, err := strconv.Atoi(pid)
		if err != nil {
			multilog.Critical("Could not convert pid string (%s) to int in heartbeat handler: %s", pid, err)
		}

		dims := &dimensions.Values{
			Trigger:          p.StrP(target.TriggerExec.String()),
			Headless:         &headless,
			CommitID:         &commit,
			ProjectNameSpace: &namespace,
			InstanceID:       p.StrP(instanceid.Make()),
		}
		dimsJSON, err := dims.Marshal()
		if err != nil {
			multilog.Critical("Could not marshal dimensions in heartbeat handler: %s", err)
		}

		_, err = reporter.RuntimeUsage(context.Background(), pidNum, exec, dimsJSON)
		if err != nil {
			multilog.Critical("Failed to report runtime usage in heartbeat handler: %s", errs.JoinMessage(err))
		}

		return "", true
	}
}

func (c *Comm) SendHeartbeat(ctx context.Context, pid string) (string, error) {
	return c.req.Request(ctx, KeyHeartbeat+pid)
}
