package downloadlogs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	LogUrl string
}

func NewParams() *Params {
	return &Params{}
}

// Example: {"body": {"facility": "INFO", "msg": "..."}, "artifact_id": "...", "timestamp": "2025-08-12T19:23:51.702971", "type": "artifact_progress", "source": "build-wrapper", "pid": 19}
type LogLine struct {
	Body struct {
		Msg string `json:"msg"`
	} `json:"body"`
}

func (runner *DownloadLogsRunner) Run(params *Params) error {
	response, err := http.Get(params.LogUrl)
	if err != nil {
		return fmt.Errorf("error while downloading logs: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("error fetching logs: status %d, %s", response.StatusCode, body)
	}

	scanner := bufio.NewScanner(response.Body)

	// Read all lines, parse and only store the text messages
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		var logLine LogLine
		if err := json.Unmarshal([]byte(line), &logLine); err != nil {
			continue
		}
		lines = append(lines, logLine.Body.Msg)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log content: %v", err)
	}

	// Check what lines contain the keyword "error" and print the previous 10 and next 10 lines
	printedLines := make(map[int]bool)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), "error") {
			start := i - 10
			if start < 0 {
				start = 0
			}
			end := i + 10
			if end >= len(lines) {
				end = len(lines) - 1
			}

			for j := start; j <= end; j++ {
				if !printedLines[j] {
					// Print ellipsis if there are skipped lines
					if j > 0 && !printedLines[j-1] {
						runner.output.Print("[...]")
					}
					runner.output.Print(lines[j])
					printedLines[j] = true
				}
			}
		}
	}

	return nil
}
