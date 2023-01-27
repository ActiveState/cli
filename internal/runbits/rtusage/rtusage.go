package rtusage

import (
	"context"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/notify"
	"github.com/ActiveState/cli/internal/output"
	"strconv"
)

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
	} else if res.Usage > res.Limit {
		out.Notice(locale.Tr("runtime_usage_limit_reached", orgName, strconv.Itoa(res.Usage), strconv.Itoa(res.Limit)))
	}
}

func NotifyRuntimeUsage(data dataHandler, orgName string) {
	if orgName == "" {
		return
	}

	usage, err := data.CheckRuntimeUsage(context.Background(), orgName)
	if err != nil {
		multilog.Error("Soft limit: Failed to check runtime usage in heartbeat handler: %s", errs.JoinMessage(err))
		return
	}

	if usage.Usage > usage.Limit {
		err := notify.Send(locale.Tl("runtime_limit_reached_title", "Runtime Limit Reached"),
			locale.Tl("runtime_limit_reached_msg", "Heads up! You've reached your runtime limit for ActiveState-Labs."),
			locale.Tl("runtime_limit_reached_action", "Upgrade"),
			"state://platform/upgrade") // We have to use the state protocol because https:// is backgrounded by the OS
		if err != nil {
			multilog.Error("Soft limit: Failed to send notification: %s", errs.JoinMessage(err))
		}
	}
}
