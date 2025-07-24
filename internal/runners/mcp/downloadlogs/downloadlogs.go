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
}

func New(p *primer.Values) *DownloadLogsRunner {
	return &DownloadLogsRunner{
		output: p.Output(),
	}
}

type Params struct {
	logUrl string
}

func NewParams(logUrl string) *Params {
	return &Params{
		logUrl: logUrl,
	}
}

func (runner *DownloadLogsRunner) Run(params *Params) error {
	response, err := http.Get(params.logUrl)
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
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}
		return fmt.Errorf("error fetching logs: status %d, %s", response.StatusCode, body)
	}
}
