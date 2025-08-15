package downloadlogs

import (
	"bufio"
	"encoding/json"
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

// Example: {"body": {"facility": "INFO", "msg": "..."}, "artifact_id": "...", "timestamp": "2025-08-12T19:23:51.702971", "type": "artifact_progress", "source": "build-wrapper", "pid": 19}
type LogLine struct {
	Body struct {
		Msg string `json:"msg"`
	} `json:"body"`
}

func (runner *DownloadLogsRunner) Run(params *Params) error {
	response, err := http.Get(params.logUrl)
	if err != nil {
		return fmt.Errorf("error while downloading logs: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("error fetching logs: status %d, %s", response.StatusCode, body)
	}

	scanner := bufio.NewScanner(response.Body)

	startPrinting := false

	for scanner.Scan() {
		var logLine LogLine
		line := scanner.Text()

		if err := json.Unmarshal([]byte(line), &logLine); err != nil {
			continue // Skip malformed lines
		}

		msg := logLine.Body.Msg

		if !startPrinting {
			if msg == "Dependencies downloaded and unpacked." {
				startPrinting = true
			}
			continue
		}

		runner.output.Print(msg + "\n")
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log content: %v", err)
	}

	return nil
}
