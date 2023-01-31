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
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/runbits/rtusage"
	"github.com/ActiveState/cli/internal/svcctl/svcmsg"
	"github.com/ActiveState/cli/pkg/platform/runtime/executors/execmeta"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
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
	ReportRuntimeUsage(ctx context.Context, pid int, exec, dimensionsJSON string) (*graph.ReportRuntimeUsageResponse, error)
	CheckRuntimeUsage(ctx context.Context, organizationName string) (*graph.CheckRuntimeUsageResponse, error)
}

type AnalyticsReporter interface {
	EventWithLabel(category string, action, label string, dims ...*dimensions.Values)
}

func HeartbeatHandler(cfg *config.Instance, resolver Resolver, analyticsReporter AnalyticsReporter) ipc.RequestHandler {
	return func(input string) (string, bool) {
		if !strings.HasPrefix(input, KeyHeartbeat) {
			return "", false
		}

		data := input[len(KeyHeartbeat):]
		hb := svcmsg.NewHeartbeatFromSvcMsg(data)

		go func() {
			defer panics.HandlePanics(recover(), debug.Stack())
			
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
				Trigger:          p.StrP(target.TriggerExec.String()),
				Headless:         p.StrP(strconv.FormatBool(metaData.Headless)),
				CommitID:         p.StrP(metaData.CommitUUID),
				ProjectNameSpace: p.StrP(metaData.Namespace),
				InstanceID:       p.StrP(instanceid.Make()),
			}
			dimsJSON, err := dims.Marshal()
			if err != nil {
				multilog.Critical("Heartbeat Failure: Could not marshal dimensions in heartbeat handler: %s", err)
				return
			}

			// Soft limit notification
			logging.Debug("Checking runtime usage for %s", metaData.Namespace)
			if metaData.Namespace != "" {
				ns, err := project.ParseNamespace(metaData.Namespace)
				if err != nil {
					multilog.Error("Soft limit: Could not parse namespace in heartbeat handler: %s", err)
				} else {
					rtusage.NotifyRuntimeUsage(cfg, resolver, ns.Owner)
				}
			}

			analyticsReporter.EventWithLabel(constants.CatRuntimeUsage, constants.ActRuntimeAttempt, "", dims)
			_, err = resolver.ReportRuntimeUsage(context.Background(), pidNum, hb.ExecPath, dimsJSON)
			if err != nil {
				multilog.Critical("Heartbeat Failure: Failed to report runtime usage in heartbeat handler: %s", errs.JoinMessage(err))
				return
			}
		}()

		return "ok", true
	}
}

func (c *Comm) SendHeartbeat(ctx context.Context, pid string) (string, error) {
	return c.req.Request(ctx, KeyHeartbeat+pid)
}
