package downloadlogs

import (
	"fmt"
	"io"
	"net/http"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
)

type DownloadLogsRunner struct {
	output output.Outputer
	logUrl string
}

func New(p *primer.Values, logUrl string) *DownloadLogsRunner {
	return &DownloadLogsRunner{
		output: p.Output(),
		logUrl: logUrl,
	}
}

func (runner *DownloadLogsRunner) Run() error {
	response, err := http.Get(runner.logUrl)
	if err != nil {
		return fmt.Errorf("error while downloading logs: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode == 200 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}
		runner.output.Print(string(body))
		return nil
	} else {
		return fmt.Errorf("wrong status code fetching logs: %d", response.StatusCode)
	}
}
