package rtusage

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/notify"
	"github.com/ActiveState/cli/internal/output"
)

const CfgKeyLastNotify = "notify.rtusage.last"

type dataHandler interface {
	CheckRuntimeUsage(ctx context.Context, organizationName string) (*graph.CheckRuntimeUsageResponse, error)
}

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

	if usage <= res.Limit {
		return
	}

	// Override silence time
	silenceMs := time.Hour.Milliseconds()
	if override := os.Getenv(constants.RuntimeUsageSilenceTimeOverrideEnvVarName); override != "" {
		overrideInt, err := strconv.ParseInt(override, 10, 64)
		if err != nil {
			logging.Error("Failed to parse runtime usage silence time override: %v", err)
		} else {
			silenceMs = overrideInt
		}
	}

	// Don't notify if we already notified recently
	if time.Now().Sub(cfg.GetTime(CfgKeyLastNotify)).Milliseconds() <= silenceMs {
		return
	}

	if err := cfg.Set(CfgKeyLastNotify, time.Now()); err != nil {
		multilog.Error("Soft limit: Failed to set last notify time: %s", errs.JoinMessage(err))
		return
	}

	err2 := notify.Send(locale.T("runtime_limit_reached_title"),
		locale.Tr("runtime_limit_reached_msg", orgName),
		locale.T("runtime_limit_reached_action"),
		"state://platform/upgrade") // We have to use the state protocol because https:// is backgrounded by the OS
	if err2 != nil {
		multilog.Error("Soft limit: Failed to send notification: %s", errs.JoinMessage(err))
	}
}
