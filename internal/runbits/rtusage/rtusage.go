package rtusage

import (
	"context"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/notify"
	"github.com/ActiveState/cli/internal/output"
	"os"
	"strconv"
	"time"
)

const CfgKeyLastNotify = "notify.rtusage.last"

type dataHandler interface {
	CheckRuntimeUsage(ctx context.Context, organizationName string) (*graph.CheckRuntimeUsageResponse, error)
}

type checkFunc func(ctx context.Context, organizationName string) (*graph.CheckRuntimeUsageResponse, error)

func PrintRuntimeUsage(data dataHandler, out output.Outputer, orgName string) {
	if orgName == "" {
		return
	}

	logging.Debug("Checking to print runtime usage for %s", orgName)

	res, err := data.CheckRuntimeUsage(context.Background(), orgName)
	if err != nil {
		// Runtime usage is not enforced, so any errors should not interrupt the user either
		multilog.Error("Could not check runtime usage: %v", errs.JoinMessage(err))
		return
	}

	usage := res.Usage
	if override := os.Getenv(constants.RuntimeUsageOverrideEnvVarName); override != "" {
		logging.Debug("Overriding usage with %s", override)
		usage, _ = strconv.Atoi(override)
	}

	if usage > res.Limit {
		out.Notice(locale.Tr("runtime_usage_limit_reached", orgName, strconv.Itoa(usage), strconv.Itoa(res.Limit)))
	}
}

func NotifyRuntimeUsage(cfg *config.Instance, data dataHandler, orgName string) {
	if orgName == "" {
		return
	}

	if time.Now().Sub(cfg.GetTime(CfgKeyLastNotify)).Minutes() < float64(60) {
		return
	}

	if err := cfg.Set(CfgKeyLastNotify, time.Now()); err != nil {
		multilog.Error("Soft limit: Failed to set last notify time: %s", errs.JoinMessage(err))
		return
	}

	res, err := data.CheckRuntimeUsage(context.Background(), orgName)
	if err != nil {
		multilog.Error("Soft limit: Failed to check runtime usage in heartbeat handler: %s", errs.JoinMessage(err))
		return
	}

	usage := res.Usage
	if override := os.Getenv(constants.RuntimeUsageOverrideEnvVarName); override != "" {
		logging.Debug("Overriding usage with %s", override)
		usage, _ = strconv.Atoi(override)
	}

	if usage > res.Limit {
		err := notify.Send(locale.T("runtime_limit_reached_title"),
			locale.T("runtime_limit_reached_msg"),
			locale.T("runtime_limit_reached_action"),
			"state://platform/upgrade") // We have to use the state protocol because https:// is backgrounded by the OS
		if err != nil {
			multilog.Error("Soft limit: Failed to send notification: %s", errs.JoinMessage(err))
		}
	}
}
