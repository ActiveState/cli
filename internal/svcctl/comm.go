package svcctl

import (
	"context"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/svcctl/svcmsg"
	"github.com/ActiveState/cli/pkg/runtime/executors/execmeta"
)

var (
	KeyHTTPAddr  = "http-addr"
	KeyLogFile   = "log-file"
	KeyHeartbeat = "heart<"
	KeyExitCode  = "exitcode<"
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
			return logFile, true
		}
		return "", false
	}
}

func (c *Comm) GetHTTPAddr(ctx context.Context) (string, error) {
	return c.req.Request(ctx, KeyHTTPAddr)
}

func (c *Comm) GetLogFileName(ctx context.Context) (string, error) {
	return c.req.Request(ctx, KeyLogFile)
}

type Resolver interface {
	ReportRuntimeUsage(ctx context.Context, pid int, exec, source string, dimensionsJSON string) (*graph.ReportRuntimeUsageResponse, error)
}

type AnalyticsReporter interface {
	EventWithSource(category, action, source string, dims ...*dimensions.Values)
	EventWithSourceAndLabel(category, action, source, label string, dims ...*dimensions.Values)
}

func HeartbeatHandler(cfg *config.Instance, resolver Resolver, analyticsReporter AnalyticsReporter) ipc.RequestHandler {
	return func(input string) (string, bool) {
		if !strings.HasPrefix(input, KeyHeartbeat) {
			return "", false
		}

		logging.Debug("Heartbeat: Received heartbeat through ipc")

		data := input[len(KeyHeartbeat):]
		hb := svcmsg.NewHeartbeatFromSvcMsg(data)

		go func() {
			defer func() { panics.HandlePanics(recover(), debug.Stack()) }()

			pidNum, err := strconv.Atoi(hb.ProcessID)
			if err != nil {
				multilog.Error("Heartbeat: Could not convert pid string (%s) to int in heartbeat handler: %s", hb.ProcessID, err)
			}

			metaFilePath := filepath.Join(filepath.Dir(hb.ExecPath), execmeta.MetaFileName)
			logging.Debug("Reading meta data from filepath (%s)", metaFilePath)
			metaData, err := execmeta.NewFromFile(metaFilePath)
			if err != nil {
				multilog.Critical("Heartbeat Failure: Could not read meta data from filepath (%s): %s", metaFilePath, err)
				return
			}

			if metaData.Namespace == "" && metaData.CommitUUID == "" {
				multilog.Critical("Heartbeat Failure: Meta data is missing namespace and commitUUID: %v", metaData)
			}

			dims := &dimensions.Values{
				Trigger:          ptr.To(trigger.TriggerExecutor.String()),
				Headless:         ptr.To(strconv.FormatBool(metaData.Headless)),
				CommitID:         ptr.To(metaData.CommitUUID),
				ProjectNameSpace: ptr.To(metaData.Namespace),
				InstanceID:       ptr.To(instanceid.Make()),
				Sequence:         ptr.To(-1), // Sequence is irrelevant for attempt / heartbeats
			}
			dimsJSON, err := dims.Marshal()
			if err != nil {
				multilog.Critical("Heartbeat Failure: Could not marshal dimensions in heartbeat handler: %s", err)
				return
			}

			logging.Debug("Firing runtime usage events for %s", metaData.Namespace)
			analyticsReporter.EventWithSource(constants.CatRuntimeUsage, constants.ActRuntimeAttempt, constants.SrcExecutor, dims)
			_, err = resolver.ReportRuntimeUsage(context.Background(), pidNum, hb.ExecPath, constants.SrcExecutor, dimsJSON)
			if err != nil {
				multilog.Critical("Heartbeat Failure: Failed to report runtime usage in heartbeat handler: %s", errs.JoinMessage(err))
				return
			}
		}()

		return "ok", true
	}
}

func ExitCodeHandler(cfg *config.Instance, resolver Resolver, analyticsReporter AnalyticsReporter) ipc.RequestHandler {
	return func(input string) (string, bool) {
		defer func() { panics.HandlePanics(recover(), debug.Stack()) }()

		if !strings.HasPrefix(input, KeyExitCode) {
			return "", false
		}

		logging.Debug("Exit Code: Received exit code through ipc")

		data := input[len(KeyExitCode):]
		exitCode := svcmsg.NewExitCodeFromSvcMsg(data)

		logging.Debug("Firing exit code event for %s", exitCode.ExecPath)
		analyticsReporter.EventWithSourceAndLabel(constants.CatDebug, constants.ActExecutorExit, constants.SrcExecutor, exitCode.ExitCode, &dimensions.Values{
			Command: ptr.To(exitCode.ExecPath),
		})

		return "ok", true
	}
}
