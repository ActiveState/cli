package camel

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
)

func loadRelocationFile(relocFilePath string) map[string]bool {
	relocBytes, err := os.ReadFile(relocFilePath)
	if err != nil {
		logging.Debug("Could not open relocation file: %v", err)
		return nil
	}
	reloc := string(relocBytes)
	relocMap := map[string]bool{}
	entries := strings.Split(reloc, "\n")
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		info := strings.Split(entry, " ")
		// Place path suffix into map
		relocMap[info[1]] = true
	}
	return relocMap
}
