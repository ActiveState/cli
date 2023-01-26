package rtusage

import (
	"context"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"strconv"
)

func ReportRuntimeUsage(svcModel *model.SvcModel, out output.Outputer, orgName string) {
	if orgName == "" {
		return
	}
	
	res, err := svcModel.CheckRuntimeUsage(context.Background(), orgName)
	if err != nil {
		// Runtime usage is not enforced, so any errors should not interrupt the user either
		multilog.Error("Could not check runtime usage: %v", errs.JoinMessage(err))
	} else if res.Usage > res.Limit {
		out.Notice(locale.Tr("runtime_usage_limit_reached", orgName, strconv.Itoa(res.Usage), strconv.Itoa(res.Limit)))
	}
}
